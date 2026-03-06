package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

var (
	activeCmd *exec.Cmd
	cmdMutex  sync.Mutex

	// State management for the build process
	buildCancel context.CancelFunc
	buildMutex  sync.Mutex
)

// runBuild now accepts a context tether.
func runBuild(ctx context.Context, commandString string) error {
	slog.Info("Initiating compilation...", "command", commandString)
	parts := strings.Split(commandString, " ")

	// exec.CommandContext links the process to our cancellation tether.
	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

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

func stopServer() {
	cmdMutex.Lock()
	defer cmdMutex.Unlock()

	if activeCmd != nil && activeCmd.Process != nil {
		slog.Info("Halting existing server process...")
		activeCmd.Process.Kill()
		activeCmd.Wait()
		activeCmd = nil
	}
}

func main() {
	rootPtr := flag.String("root", ".", "Directory to watch")
	buildPtr := flag.String("build", "", "Command to build")
	execPtr := flag.String("exec", "", "Command to run")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if *buildPtr == "" || *execPtr == "" {
		slog.Error("Both --build and --exec commands are strictly required.")
		os.Exit(1)
	}

	// Initial synchronous build using a background (uncancellable) context
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

	if err := watcher.Add(*rootPtr); err != nil {
		slog.Error("Failed to watch directory", "error", err)
		os.Exit(1)
	}

	slog.Info("Observer active. Watching for changes...", "directory", *rootPtr)

	var rebuildTimer *time.Timer
	debounceDelay := 500 * time.Millisecond

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Rename) {
				if rebuildTimer != nil {
					rebuildTimer.Stop()
				}

				rebuildTimer = time.AfterFunc(debounceDelay, func() {
					stopServer()

					// Preemption Logic
					buildMutex.Lock()
					if buildCancel != nil {
						slog.Info("Preempting previous compilation...")
						buildCancel() // Sever the tether
					}

					// Forge a new tether for the upcoming build
					var ctx context.Context
					ctx, buildCancel = context.WithCancel(context.Background())
					buildMutex.Unlock()

					// Execute the build with the new tether
					if err := runBuild(ctx, *buildPtr); err != nil {
						// If the error was our own intentional cancellation, remain calm.
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
