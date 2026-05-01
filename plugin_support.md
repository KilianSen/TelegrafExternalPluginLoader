# Telegraf Plugin Support Guidelines

To ensure your external Telegraf plugin is compatible with the **Telegraf External Plugin Loader**, please follow these guidelines.

## 1. Supported Source Types

The loader identifies sources based on their URL structure:

- **Git Repositories**: URLs ending in `.git` (e.g., `https://github.com/user/my-plugin.git`). These will be cloned and built from source.
- **Direct Binaries**: URLs pointing directly to a file (e.g., `https://example.com/downloads/my-plugin`). These will be downloaded and marked as executable.

## 2. Build Requirements (for Git Repos)

If you are providing a source repository, the loader supports two build methods:

### Method A: Makefile (Preferred)
If a `Makefile` is present in the root of your repository, the loader will execute the `make` command. 
- Ensure the default target (or `all`) builds the plugin.
- The resulting binary should be placed in the root directory or a `build/` subdirectory.

### Method B: Go Build (Fallback)
If no `Makefile` is found, the loader will attempt to build using:
```bash
go mod tidy
go build -o plugin_binary .
```
- Your repository **must** contain a `go.mod` file.
- The main package should be in the root directory.

## 3. Binary Detection & Naming

After building, the loader scans for the resulting executable. To make detection reliable:

- **Naming**: The loader prefers a binary that matches your repository name (e.g., if the repo is `telegraf-my-plugin.git`, it looks for `telegraf-my-plugin`).
- **Extensions**: Avoid using extensions like `.go`, `.sh`, `.py`, `.md`, or `.txt` for your final binary, as the loader ignores these to prevent picking up source files.
- **Permissions**: The loader will automatically set the executable bit (`chmod +x`) on the final file.

## 4. Example Repository Structure

A well-structured Go-based plugin repository should look like this:

```text
my-telegraf-plugin/
├── go.mod          # Required for Go plugins
├── go.sum          # Recommended
├── main.go         # Entry point
├── Makefile        # Highly Recommended
├── README.md       # Documentation
└── (other source files)
```

### Recommended Makefile
```makefile
all: build

build:
	go build -o my-telegraf-plugin .

clean:
	rm -f my-telegraf-plugin
```

## 5. Execution Environment

- **Architecture**: The loader currently runs in a Linux Alpine environment (`golang:alpine`). Ensure your plugin is compatible with Alpine (consider CGO dependencies; the loader includes `gcc` and `musl-dev` to help).
- **Paths**: Plugins are moved to the directory specified by `PLUGIN_DIR` (default: `/plugins`). Your Telegraf configuration should point to this directory.
