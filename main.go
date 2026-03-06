package main

import (
	"flag"
	"log/slog"
	"os"
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
}
