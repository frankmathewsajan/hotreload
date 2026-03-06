package main

import (
	"flag"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Global state to track and kill the running server
var (
	activeCmd *exec.Cmd
	cmdMutex  sync.Mutex
)

func runBuild(commandString string) error {
	slog.Info("Initiating compilation...", "command", commandString)
	parts := strings.Split(commandString, " ")
	cmd := exec.Command(parts[0], parts[1:]...)
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
	// 1. Define and parse flags
	rootPtr := flag.String("root", ".", "Directory to watch for file changes")
	buildPtr := flag.String("build", "", "Command used to build the project")
	execPtr := flag.String("exec", "", "Command used to run the built server")
	flag.Parse()

	// 2. Configure logging
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if *buildPtr == "" || *execPtr == "" {
		slog.Error("Both --build and --exec commands are strictly required.")
		os.Exit(1)
	}

	slog.Info("Hot Reload Engine Initialized",
		"root", *rootPtr,
		"build", *buildPtr,
		"exec", *execPtr,
	)

	// Phase 1: The Initial Build
	if err := runBuild(*buildPtr); err != nil {
		slog.Error("Primary compilation failed. Halting.", "error", err)
		os.Exit(1)
	}

	// Phase 2: The Initial Ignition
	if err := startServer(*execPtr); err != nil {
		slog.Error("Failed to start server.", "error", err)
		os.Exit(1)
	}

	// Phase 3: The Observer Setup
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		slog.Error("Failed to initialize file watcher", "error", err)
		os.Exit(1)
	}
	defer watcher.Close()

	err = watcher.Add(*rootPtr)
	if err != nil {
		slog.Error("Failed to watch directory", "root", *rootPtr, "error", err)
		os.Exit(1)
	}

	slog.Info("Observer active. Watching for changes...", "directory", *rootPtr)

	// Phase 4: The Debounce Mechanism
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

				slog.Info("File event registered. Awaiting stability...", "file", event.Name)

				rebuildTimer = time.AfterFunc(debounceDelay, func() {
					slog.Info("File system stable. Commencing reload sequence.")

					stopServer()

					if err := runBuild(*buildPtr); err == nil {
						startServer(*execPtr)
					} else {
						slog.Error("Recompilation failed. Waiting for next file change.")
					}
				})
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			slog.Error("Watcher encountered an error", "error", err)
		}
	}
}
