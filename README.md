# Telegraf External Plugin Loader

This tool loads external plugins for Telegraf from specified sources and places them in the `/plugins` directory.

## Usage

Set the `PLUGIN_SOURCES` environment variable to a comma-separated list of sources.

Sources can be:
- Git repositories ending in `.git` (e.g., `https://github.com/user/plugin.git`): The repo is cloned, `make` is run to build, and the resulting executable is copied.
- Direct file URLs (e.g., `https://example.com/plugin`): The file is downloaded and made executable.

Example:
```bash
docker run -e PLUGIN_SOURCES="https://github.com/user/plugin1.git,https://example.com/plugin2" -v /host/plugins:/plugins ghcr.io/kiliansen/telegraf-plugin-loader:latest
```

## Building

```bash
go build -o plugin-loader main.go
```

Or use Docker:
```bash
docker build -t telegraf-plugin-loader .
```

## Docker Compose Example

See `docker-compose.yml` for an example setup that runs the plugin loader and then Telegraf with the loaded plugins.

## CI/CD

On push to main, the image is automatically built and published to GitHub Container Registry with `latest` and the next semantic version tag based on conventional commits.
