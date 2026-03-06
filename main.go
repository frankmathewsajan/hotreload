package main

import (
	"flag"
	"log/slog"
	"os"
	"os/exec"
	"strings"
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
	slog.Info("Initiating primary build sequence...")
	if err := executeCommand(*buildPtr); err != nil {
		slog.Error("Primary compilation failed. Halting.", "error", err)
		os.Exit(1)
	}

	// Phase 2: The Initial Ignition
	slog.Info("Build successful. Igniting server...")
	if err := executeCommand(*execPtr); err != nil {
		slog.Error("Server execution terminated abruptly.", "error", err)
	}
}

// Add "strings" and "os/exec" to your import block at the top.

// executeCommand acts as a universal wrapper for running terminal commands.
func executeCommand(commandString string) error {
	// Assumption: We assume arguments are separated by spaces.
	// A robust shell parser would be needed for complex quoted arguments,
	// but this suffices for the assignment's expected inputs.
	parts := strings.Split(commandString, " ")
	mainCommand := parts[0]
	args := parts[1:]

	// Construct the command architecture.
	cmd := exec.Command(mainCommand, args...)

	// CRITICAL REQUIREMENT: Stream logs in real-time.
	// By attaching the child's output directly to our operating system's standard output,
	// we bypass any Go-level buffering.
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	slog.Info("Executing process", "command", commandString)

	// Run executes the command and waits for it to complete.
	return cmd.Run()
}
