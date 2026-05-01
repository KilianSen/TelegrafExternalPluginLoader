# Telegraf External Plugin Loader

This project provides a Go-based utility designed to download, build, and manage external plugins for Telegraf.

## Project Overview

- **Core Functionality**: Downloads plugins from Git repositories or direct URLs, builds them, and places the executables in a designated plugin directory.
- **Main Technologies**: Go (1.25+), Docker, GitHub Actions (for CI/CD).
- **Architecture**: A concurrent Go application that handles parallel downloads and builds.

## Building and Running

### Local Development
To build and test:
```bash
go build -o plugin-loader main.go
go test ./...
```

### Docker
The Docker image is based on `golang:alpine` and includes `git`, `make`, `gcc`, and `musl-dev` to support CGO-based plugins.
```bash
docker build -t telegraf-plugin-loader .
```

To run:
```bash
docker run -e PLUGIN_SOURCES="..." -e PLUGIN_DIR="/custom/plugins" -v /path:/custom/plugins telegraf-plugin-loader
```

## Development Conventions

- **Concurrency**: Processes multiple sources in parallel using Go routines and `sync.WaitGroup`.
- **Environment Variables**:
    - `PLUGIN_SOURCES`: Comma-separated list of Git URLs or direct file links.
    - `PLUGIN_DIR`: Directory where plugins are stored (defaults to `/plugins`).
- **Build Logic**:
    - Uses `make` if a `Makefile` is present.
    - Falls back to `go build .` if no `Makefile` exists.
- **Binary Detection**: 
    - Heuristically identifies binaries by checking executable bits and excluding source/documentation extensions.
    - Robustly parses filenames from URLs, ignoring query parameters.
- **Error Handling**: Properly handles and returns errors from file operations and subprocesses.

## Key Files
- `main.go`: Application logic.
- `main_test.go`: Unit tests for core logic.
- `Dockerfile`: Container definition with build dependencies.
- `docker-compose.yml`: Example integration with Telegraf.
