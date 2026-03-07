
all: test run

test:
	go test -v ./...

run:
	go run main.go --root ./tserver --build "go build -o ./bin/server.exe ./tserver/main.go" --exec "./bin/server.exe"

clean:
	rm -rf ./bin