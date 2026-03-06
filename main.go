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
	activeCmd   *exec.Cmd
	cmdMutex    sync.Mutex
	buildCancel context.CancelFunc
	buildMutex  sync.Mutex
)

// runBuild compiles the target binary.
func runBuild(ctx context.Context, commandString string) error {
	slog.Info("Initiating compilation...", "command", commandString)
	parts := strings.Split(commandString, " ")
	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// startServer ignites the compiled binary.
func startServer(commandString string) error {
	cmdMutex.Lock()
	defer cmdMutex.Unlock()

	slog.Info("Igniting server...", "command", commandString)
	parts := strings.Split(commandString, " ")
	activeCmd = exec.Command(parts[0], parts[1:]...)
	activeCmd.Stdout = os.Stdout
	activeCmd.Stderr = os.Stderr

	return activeCmd.Start()
}

// stopServer executes a Graceful-to-Ruthless termination sequence.
func stopServer() {
	cmdMutex.Lock()
	defer cmdMutex.Unlock()

	if activeCmd != nil && activeCmd.Process != nil {
		slog.Info("Commencing server shutdown sequence...")

		done := make(chan error, 1)
		go func() {
			done <- activeCmd.Wait()
		}()

		err := activeCmd.Process.Signal(os.Interrupt)
		if err != nil {
			slog.Warn("Graceful interrupt unsupported or failed. Preparing to use force.")
		} else {
			slog.Info("Interrupt signal sent. Awaiting cooperative termination...")
		}

		select {
		case <-time.After(3 * time.Second):
			slog.Warn("Server is stubborn. Executing ruthless termination.")
			activeCmd.Process.Kill()
			<-done
			slog.Info("Stubborn server eradicated.")
		case err := <-done:
			if err != nil {
				slog.Info("Server terminated with an exit code.", "error", err)
			} else {
				slog.Info("Server terminated gracefully.")
			}
		}

		activeCmd = nil
	}
}

// NEW: Recursive Directory Watcher with Filtering
func watchRecursive(watcher *fsnotify.Watcher, root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// We only add directories to the watcher
		if d.IsDir() {
			name := d.Name()
			// THE FILTER: Ignore heavy, irrelevant, or temporary directories.
			if name == ".git" || name == "node_modules" || name == "bin" || name == "tmp" || name == "vendor" {
				slog.Debug("Ignoring directory", "dir", path)
				return filepath.SkipDir // Do not look inside this folder
			}

			err := watcher.Add(path)
			if err != nil {
				slog.Error("Failed to watch sub-directory", "path", path, "error", err)
			} else {
				slog.Info("Added to watch list", "directory", path)
			}
		}
		return nil
	})
}

func main() {
	rootPtr := flag.String("root", ".", "Directory to watch")
	buildPtr := flag.String("build", "", "Command to build")
	execPtr := flag.String("exec", "", "Command to run")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	if *buildPtr == "" || *execPtr == "" {
		slog.Error("Both --build and --exec commands are strictly required.")
		os.Exit(1)
	}

	if err := runBuild(context.Background(), *buildPtr); err != nil {
		slog.Error("Primary compilation failed.", "error", err)
		os.Exit(1)
	}

	if err := startServer(*execPtr); err != nil {
		slog.Error("Failed to start server.", "error", err)
		os.Exit(1)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		slog.Error("Failed to initialize watcher", "error", err)
		os.Exit(1)
	}
	defer watcher.Close()

	// NEW: Use our recursive function instead of watcher.Add()
	if err := watchRecursive(watcher, *rootPtr); err != nil {
		slog.Error("Failed to perform initial directory scan", "error", err)
		os.Exit(1)
	}

	slog.Info("Observer active. Watching for changes...", "root", *rootPtr)

	var rebuildTimer *time.Timer
	debounceDelay := 500 * time.Millisecond

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			// NEW: Dynamic Directory Detection
			// If a new item is created, check if it's a directory. If so, watch it!
			if event.Has(fsnotify.Create) {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					slog.Info("New directory detected. Expanding surveillance...", "path", event.Name)
					watchRecursive(watcher, event.Name)
				}
			}

			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Rename) {

				// Optional: Filter out editor swap files or temporary files so they don't trigger rebuilds
				if strings.HasSuffix(event.Name, "~") || strings.HasPrefix(filepath.Base(event.Name), ".") {
					continue
				}

				if rebuildTimer != nil {
					rebuildTimer.Stop()
				}

				rebuildTimer = time.AfterFunc(debounceDelay, func() {
					slog.Info("File event registered. Awaiting stability...", "file", event.Name)
					stopServer()

					buildMutex.Lock()
					if buildCancel != nil {
						slog.Info("Preempting previous compilation...")
						buildCancel()
					}
					var ctx context.Context
					ctx, buildCancel = context.WithCancel(context.Background())
					buildMutex.Unlock()

					if err := runBuild(ctx, *buildPtr); err != nil {
						if ctx.Err() == context.Canceled {
							slog.Info("Prior build successfully aborted.")
						} else {
							slog.Error("Recompilation failed.", "error", err)
						}
					} else {
						startServer(*execPtr)
					}
				})
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			slog.Error("Watcher error", "error", err)
		}
	}
}
