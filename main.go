package main

import (
	"context"
	"flag"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

var (
	srv       *exec.Cmd
	srvMu     sync.Mutex
	bldCancel context.CancelFunc
	bldMu     sync.Mutex
)

// prep wires a command to the terminal output
func prep(ctx context.Context, cmdStr string) *exec.Cmd {
	parts := strings.Split(cmdStr, " ")
	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd
}

// build executes the compilation step synchronously
func build(ctx context.Context, cmdStr string) error {
	slog.Info("Compiling...", "cmd", cmdStr)
	return prep(ctx, cmdStr).Run()
}

// start ignites the server asynchronously
func start(cmdStr string) error {
	srvMu.Lock()
	defer srvMu.Unlock()
	slog.Info("Starting server...", "cmd", cmdStr)

	srv = prep(context.Background(), cmdStr)
	return srv.Start()
}

// stop performs a graceful interrupt, falling back to a ruthless kill
func stop() {
	srvMu.Lock()
	defer srvMu.Unlock()

	if srv == nil || srv.Process == nil {
		return
	}

	// chan struct{} is idiomatic Go for a signal-only channel
	done := make(chan struct{})
	go func() { srv.Wait(); close(done) }()

	_ = srv.Process.Signal(os.Interrupt)

	select {
	case <-time.After(100 * time.Millisecond):
		srv.Process.Kill()
		<-done
	case <-done:
	}
	srv = nil
}

// watch recursively adds directories to the watcher, filtering noise
func watch(w *fsnotify.Watcher, root string) error {
	return filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return err
		}
		switch d.Name() {
		case ".git", "node_modules", "bin", "tmp", "vendor":
			return filepath.SkipDir
		}
		return w.Add(p)
	})
}

// dieIf is an idiomatic helper to reduce boilerplate during startup
func dieIf(err error, msg string) {
	if err != nil {
		slog.Error(msg, "err", err)
		os.Exit(1)
	}
}

func main() {
	root := flag.String("root", ".", "Directory to watch")
	bldCmd := flag.String("build", "", "Command to build")
	runCmd := flag.String("exec", "", "Command to run")
	flag.Parse()

	if *bldCmd == "" || *runCmd == "" {
		slog.Error("Flags --build and --exec required")
		os.Exit(1)
	}

	dieIf(build(context.Background(), *bldCmd), "Initial build failed")
	dieIf(start(*runCmd), "Initial start failed")

	w, err := fsnotify.NewWatcher()
	dieIf(err, "Watcher init failed")
	defer w.Close()

	dieIf(watch(w, *root), "Directory scan failed")
	slog.Info("Engine active", "root", *root)

	var timer *time.Timer
	for {
		select {
		case e, ok := <-w.Events:
			if !ok {
				return
			}

			// Dynamic directory detection
			if e.Has(fsnotify.Create) {
				if i, err := os.Stat(e.Name); err == nil && i.IsDir() {
					watch(w, e.Name)
				}
			}

			// File extension filter
			if e.Has(fsnotify.Write) || e.Has(fsnotify.Create) || e.Has(fsnotify.Rename) {
				ext := filepath.Ext(e.Name)
				if ext != ".go" && ext != ".mod" && ext != ".sum" {
					continue
				}

				if timer != nil {
					timer.Stop()
				}

				// The Debounce & Preemption Payload
				timer = time.AfterFunc(500*time.Millisecond, func() {
					stop()

					bldMu.Lock()
					if bldCancel != nil {
						bldCancel()
					}
					ctx, cancel := context.WithCancel(context.Background())
					bldCancel = cancel
					bldMu.Unlock()

					if err := build(ctx, *bldCmd); err == nil {
						start(*runCmd)
					} else if ctx.Err() != context.Canceled {
						slog.Error("Build failed", "err", err)
					}
				})
			}
		case err, ok := <-w.Errors:
			if !ok {
				return
			}
			slog.Error("Watcher error", "err", err)
		}
	}
}
