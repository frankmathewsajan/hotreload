
all: test run

test:
	go test -v ./...

run:
	go run main.go --root ./testserver --build "go build -o ./bin/server.exe ./testserver/main.go" --exec "./bin/server.exe"

clean:
	rm -rf ./bin