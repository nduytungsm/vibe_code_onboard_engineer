# Repo Explanation Tool

A Golang application that helps onboard new developers by analyzing and explaining codebases using LLM integration.

## Features

- **Dual Mode Application**: Run as either a web API server or interactive CLI
- **Web API**: RESTful API with health check endpoint
- **Interactive CLI**: REPL-style command-line interface
- **MVC Architecture**: Clean separation of concerns
- **Production Ready**: Built with Echo framework and proper error handling

## Project Structure

```
├── cmd/                    # Standalone entry points
│   ├── server/            # Standalone server binary
│   └── cli/               # Standalone CLI binary
├── controllers/           # HTTP request handlers
├── routes/               # Route definitions
├── models/               # Data models (future use)
├── cli/                  # CLI/REPL functionality
├── bin/                  # Compiled binaries
├── main.go              # Main entry point with mode selection
└── Makefile             # Build and run commands
```

## Building the Application

### Using Makefile (Recommended)
```bash
# Build all binaries
make build

# Build specific components
make build-server
make build-cli
```

### Manual Build
```bash
# Build main application (dual mode)
go build -o bin/repo-explanation .

# Build standalone binaries
go build -o bin/server cmd/server/main.go
go build -o bin/cli cmd/cli/main.go
```

## Running the Application

### Server Mode (API)
```bash
# Using main binary with flag
./bin/repo-explanation -mode=server
# OR using Makefile
make run-server
# OR using standalone binary
./bin/server
```

The server runs on `http://localhost:8080`

**Available Endpoints:**
- `GET /health` - Health check endpoint

Example:
```bash
curl http://localhost:8080/health
# Response: {"message":"Server is running","service":"repo-explanation","status":"healthy"}
```

### CLI Mode (Interactive REPL)
```bash
# Using main binary with flag
./bin/repo-explanation -mode=cli
# OR using Makefile
make run-cli
# OR using standalone binary
./bin/cli
```

**Available CLI Commands:**
- `try me` → Returns: "i am here"
- `/end` → Gracefully exits the CLI
- Any other command → Returns: "unsupported function"

Example session:
```
🚀 Repo Explanation CLI Started
Type 'try me' to test, '/end' to exit
> try me
i am here
> hello world
unsupported function
> /end
Goodbye! 👋
```

## Development

### Dependencies
- Go 1.23.0+
- Echo v4.13.4 (web framework)

### Install Dependencies
```bash
go mod tidy
```

### Testing
```bash
# Run automated CLI test
./test_cli.sh

# Manual testing
make run-cli
```

### Clean Build Artifacts
```bash
make clean
```

## Architecture

The application follows a clean MVC architecture with dual-mode support:

1. **Controllers**: Handle HTTP requests and business logic
2. **Routes**: Define API endpoints and routing
3. **Models**: Data structures (prepared for future LLM integration)
4. **CLI**: Interactive command-line interface with REPL functionality

The design ensures that both server and CLI modes can coexist without interference, allowing flexible deployment options.
