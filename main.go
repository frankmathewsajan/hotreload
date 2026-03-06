package main

import (
	"flag"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"
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

	// For now, we need to prevent the main program from exiting immediately
	// since startServer no longer blocks. We will replace this in the next commit.
	select {}
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
