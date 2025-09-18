# Makefile for repo-explanation project

.PHONY: build build-server build-cli run-server run-cli clean test

# Default target
all: build

# Build both server and CLI
build:
	go build -o bin/repo-explanation .
	go build -o bin/server cmd/server/main.go
	go build -o bin/cli cmd/cli/main.go

# Build server only
build-server:
	go build -o bin/server cmd/server/main.go

# Build CLI only  
build-cli:
	go build -o bin/cli cmd/cli/main.go

# Run server (default mode)
run-server:
	go run . -mode=server

# Run CLI
run-cli:
	go run . -mode=cli

# Run server using main with flag
server:
	./bin/repo-explanation -mode=server

# Run CLI using main with flag  
cli:
	./bin/repo-explanation -mode=cli

# Run standalone server binary
server-standalone:
	./bin/server

# Run standalone CLI binary
cli-standalone:
	./bin/cli

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f server

# Test
test:
	go test ./...

# Install dependencies
deps:
	go mod tidy
	go mod download
