# Hotreload

Hotreload is a lightweight, Go-based development utility that automatically rebuilds and restarts your application when source files change. It minimizes downtime during local development by providing a fast, configurable, and reliable hot-reload loop.

## Features

- **Recursive Watching**: Monitors a target directory and its subdirectories.
- **Smart Filtering**: Automatically ignores non-source directories like .git, 
ode_modules, bin, 	mp, and vendor.
- **Targeted Triggers**: Reacts only to changes in .go, .mod, and .sum files.
- **Debouncing**: Batches rapid file events within a 500ms window to prevent redundant builds.
- **Graceful Shutdown**: Attempts to terminate the running process gracefully via os.Interrupt before falling back to a forced kill.
- **Command-Driven**: Fully configurable build and execution commands so you can drop it into any workflow.

## Prerequisites

- Go installed (1.20+ recommended)
- Write permissions for the target output directory (e.g., ./bin)

## Usage

Run the hotreload utility using the following command structure:

`bash
go run main.go --root <watch-directory> --build "<build-command>" --exec "<run-command>"
`

### Command-Line Flags

- --root: The root directory to watch for file changes (e.g., ./ or ./src).
- --build: The exact compilation command to execute when changes are detected.
- --exec: The command to start the application after a successful build.

### Quick Start Example

To watch the local ./testserver directory, compile it to ./bin/server.exe, and run it:

`bash
go run main.go --root ./testserver --build "go build -o ./bin/server.exe ./testserver/main.go" --exec "./bin/server.exe"
`
*(Note: On Windows, use the .exe extension for the compiled binary as shown above.)*

## Workflow Lifecycle

1. Parses the provided flags and executes the initial build.
2. If successful, starts the server process.
3. Initializes an fsnotify watcher on the specified root directory.
4. Upon detecting a file write or creation (and waiting 500ms to debounce):
5. Shuts down the current server process cleanly, or kills it forcefully if it hangs.
6. Cancels any stale in-flight builds.
7. Executes the newly triggered build.
8. Restarts the server upon build success. (If the build fails, the error is logged and no process starts).

## Repository Structure

- main.go: The core hot-reload engine.
- main_test.go: Unit tests validating directory and file extension filtering logic.
- 	estserver/: A sample standalone Go application for testing the watcher.
- Makefile: Convenience commands (make test, make run).
