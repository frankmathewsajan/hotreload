package main

import (
	"flag"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
)

func main() {
	// 1. Define the command-line flags.
	// The flag package requires a name, a default value, and a description.
	rootPtr := flag.String("root", ".", "Directory to watch for file changes")
	buildPtr := flag.String("build", "", "Command used to build the project")
	execPtr := flag.String("exec", "", "Command used to run the built server")

	// 2. Parse the arguments provided by the user.
	flag.Parse()

	// 3. Configure structured logging using standard log/slog.
	// We use a TextHandler to output clean, readable logs to the standard output.
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 4. Validate the inputs. If the user doesn't tell us how to build or run, we must abort.
	if *buildPtr == "" || *execPtr == "" {
		slog.Error("Both --build and --exec commands are strictly required.")
		os.Exit(1) // Exit code 1 indicates a failure to the operating system.
	}

	slog.Info("Hot Reload Engine Initialized",
		"root", *rootPtr,
		"build", *buildPtr,
		"exec", *execPtr,
	)

	slog.Info("Hot Reload Engine Initialized" /* ... */)

	// Phase 1: The Initial Build
	if err := runBuild(*buildPtr); err != nil {
		slog.Error("Primary compilation failed. Halting.", "error", err)
		os.Exit(1)
	}

	// Phase 2: The Initial Ignition (Now running in the background)
	if err := startServer(*execPtr); err != nil {
		slog.Error("Failed to start server.", "error", err)
		os.Exit(1)
	}

	// Phase 3: The Observer
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		slog.Error("Failed to initialize file watcher", "error", err)
		os.Exit(1)
	}
	defer watcher.Close()

	// Instruct the watcher to monitor our target directory.
	// Note: This only watches the top-level folder for now. We will add deep traversal later.
	err = watcher.Add(*rootPtr)
	if err != nil {
		slog.Error("Failed to watch directory", "root", *rootPtr, "error", err)
		os.Exit(1)
	}

	slog.Info("Observer active. Watching for changes...", "directory", *rootPtr)

	// The Infinite Observation Loop
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			// We only care if a file was written to or created.
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
				slog.Info("Modification detected!", "file", event.Name)

				// 1. Terminate the old server
				stopServer()

				// 2. Recompile the code
				if err := runBuild(*buildPtr); err == nil {
					// 3. Ignite the new server
					startServer(*execPtr)
				} else {
					slog.Error("Recompilation failed. Waiting for next file change to try again.")
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			slog.Error("Watcher encountered an error", "error", err)
		}
	}
}

// Add "sync" to your import block.

var (
	activeCmd *exec.Cmd
	cmdMutex  sync.Mutex // Protects activeCmd from race conditions
)

// runBuild runs synchronously. It blocks until the build is complete.
func runBuild(commandString string) error {
	slog.Info("Initiating compilation...", "command", commandString)
	parts := strings.Split(commandString, " ")
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// startServer runs asynchronously. It starts the process and returns immediately.
func startServer(commandString string) error {
	cmdMutex.Lock()
	defer cmdMutex.Unlock()

	slog.Info("Igniting server...", "command", commandString)
	parts := strings.Split(commandString, " ")
	activeCmd = exec.Command(parts[0], parts[1:]...)
	activeCmd.Stdout = os.Stdout
	activeCmd.Stderr = os.Stderr

	// Use Start() instead of Run(). Start() does not wait for the process to finish.
	return activeCmd.Start()
}

// stopServer brutally terminates the currently running server to free up the port.
func stopServer() {
	cmdMutex.Lock()
	defer cmdMutex.Unlock()

	if activeCmd != nil && activeCmd.Process != nil {
		slog.Info("Halting existing server process...")
		activeCmd.Process.Kill() // Deliver the fatal blow
		activeCmd.Wait()         // Wait for the operating system to clean up the corpse
		activeCmd = nil
	}
}
