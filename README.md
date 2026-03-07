#Hotreload

What
- Small helper to auto-build+restart a Go server during edits.

Prereqs
- Go installed (1.20+)
- `./bin` writable (create if missing)

Run
- Use this command:
  `go run main.go --root ./testserver --build "go build -o ./bin/server.exe ./testserver/main.go" --exec "./bin/server.exe"`

How it works
- Watches files under `./testserver`.
- On change: runs the `--build` command; if build succeeds, kills previous run and starts `--exec`.

Notes
- On Windows build to `.exe` as shown. Adjust paths/flags as needed.
- Stop with Ctrl+C.
