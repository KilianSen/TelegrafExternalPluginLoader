package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	// Output directory for plugins (mapped to volume in Docker)
	pluginsDir = "/plugins"
	envSources = "PLUGIN_SOURCES"
)

func main() {
	sourcesEnv := os.Getenv(envSources)
	if sourcesEnv == "" {
		fmt.Println("No PLUGIN_SOURCES environment variable found. Exiting.")
		return
	}

	// Ensure output directory exists
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		fmt.Printf("Error creating plugins directory: %v\n", err)
		os.Exit(1)
	}

	sources := strings.Split(sourcesEnv, ",")
	for _, source := range sources {
		source = strings.TrimSpace(source)
		if source == "" {
			continue
		}

		fmt.Printf("Processing source: %s\n", source)
		if err := processSource(source); err != nil {
			fmt.Printf("Error processing %s: %v\n", source, err)
		} else {
			fmt.Printf("Successfully installed %s\n", source)
		}
	}
}

func processSource(source string) error {
	// Heuristic: If it ends in .git, treat as repo. Otherwise, treat as direct file download.
	if strings.HasSuffix(source, ".git") {
		return handleRepo(source)
	}
	return handleFile(source)
}

func handleFile(url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Printf("Error closing response body: %v\n", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	fileName := filepath.Base(url)
	destPath := filepath.Join(pluginsDir, fileName)

	return writeExecutable(destPath, resp.Body)
}

func handleRepo(repoURL string) error {
	// Create temp dir for cloning
	tempDir, err := os.MkdirTemp("", "telegraf-plugin-build-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			fmt.Printf("Error removing temp dir %s: %v\n", path, err)
		}
	}(tempDir)

	// 1. Git Clone
	fmt.Println("  - Cloning repository...")
	cmd := exec.Command("git", "clone", "--depth", "1", repoURL, tempDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git clone failed: %s", string(output))
	}

	// 2. Run Make
	fmt.Println("  - Running make...")
	cmd = exec.Command("make")
	cmd.Dir = tempDir
	// Redirect stdout/stderr to see build logs in docker logs
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("make failed: %w", err)
	}

	// 3. Find Executable
	// Strategy: Look for a file named after the repo, or scan for new executable files.
	repoName := strings.TrimSuffix(filepath.Base(repoURL), ".git")

	var binPath string
	// Check for exact match first
	if isExecutable(filepath.Join(tempDir, repoName)) {
		binPath = filepath.Join(tempDir, repoName)
	} else {
		// Walk to find a likely binary (executable, not a directory, no extension)
		_ = filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
			if binPath != "" || err != nil || info.IsDir() {
				return nil
			}
			if strings.Contains(path, ".git") {
				return filepath.SkipDir
			}

			// Check executable bit and ensure no extension (avoids .sh, .go, etc)
			if info.Mode()&0111 != 0 && !strings.Contains(info.Name(), ".") {
				binPath = path
			}
			return nil
		})
	}

	if binPath == "" {
		return fmt.Errorf("no executable binary found after build")
	}

	// 4. Move to plugins folder
	destPath := filepath.Join(pluginsDir, filepath.Base(binPath))
	f, err := os.Open(binPath)
	if err != nil {
		return err
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			fmt.Printf("Error closing file %s: %v\n", binPath, err)
		}
	}(f)

	return writeExecutable(destPath, f)
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Mode()&0111 != 0
}

func writeExecutable(path string, r io.Reader) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func(out *os.File) {
		err := out.Close()
		if err != nil {
			fmt.Printf("Error closing file %s: %v\n", path, err)
		}
	}(out)

	if _, err := io.Copy(out, r); err != nil {
		return err
	}
	return os.Chmod(path, 0755)
}
