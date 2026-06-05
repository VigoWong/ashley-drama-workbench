.PHONY: test server cli build
test:
	go test ./...
server:
	go run ./cmd/server
cli:
	go run ./cmd/cli $(ARGS)
build:
	go build -o bin/server ./cmd/server && go build -o bin/cli ./cmd/cli
