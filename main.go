package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

const (
	defaultPluginsDir = "/plugins"
	envSources        = "PLUGIN_SOURCES"
	envPluginsDir     = "PLUGIN_DIR"
)

func main() {
	sourcesEnv := os.Getenv(envSources)
	if sourcesEnv == "" {
		fmt.Println("No PLUGIN_SOURCES environment variable found. Exiting.")
		return
	}

	pluginsDir := getPluginsDir()

	// Ensure output directory exists
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		fmt.Printf("Error creating plugins directory %s: %v\n", pluginsDir, err)
		os.Exit(1)
	}

	sources := strings.Split(sourcesEnv, ",")
	var wg sync.WaitGroup

	for _, source := range sources {
		source = strings.TrimSpace(source)
		if source == "" {
			continue
		}

		wg.Add(1)
		go func(s string) {
			defer wg.Done()
			fmt.Printf("Processing source: %s\n", s)
			if err := processSource(s, pluginsDir); err != nil {
				fmt.Printf("Error processing %s: %v\n", s, err)
			} else {
				fmt.Printf("Successfully installed %s\n", s)
			}
		}(source)
	}

	wg.Wait()
}

func getPluginsDir() string {
	if pd := os.Getenv(envPluginsDir); pd != "" {
		return pd
	}
	return defaultPluginsDir
}

func processSource(source string, pluginsDir string) error {
	// Heuristic: If it ends in .git, treat as repo. Otherwise, treat as direct file download.
	if strings.HasSuffix(strings.ToLower(source), ".git") {
		return handleRepo(source, pluginsDir)
	}
	return handleFile(source, pluginsDir)
}

func getFileNameFromURL(rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	name := filepath.Base(parsed.Path)
	if name == "." || name == "/" {
		return "", fmt.Errorf("could not determine filename from URL path: %s", parsed.Path)
	}
	return name, nil
}

func handleFile(sourceURL string, pluginsDir string) error {
	resp, err := http.Get(sourceURL)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	fileName, err := getFileNameFromURL(sourceURL)
	if err != nil {
		return err
	}
	destPath := filepath.Join(pluginsDir, fileName)

	return writeExecutable(destPath, resp.Body)
}

func handleRepo(repoURL string, pluginsDir string) error {
	tempDir, err := os.MkdirTemp("", "telegraf-plugin-build-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	fmt.Printf("  [%s] Cloning repository...\n", repoURL)
	cmd := exec.Command("git", "clone", "--depth", "1", repoURL, tempDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git clone failed: %s", string(output))
	}

	// Determine build method
	if _, err := os.Stat(filepath.Join(tempDir, "Makefile")); err == nil {
		fmt.Printf("  [%s] Running make...\n", repoURL)
		buildCmd := exec.Command("make")
		buildCmd.Dir = tempDir
		buildCmd.Stdout = os.Stdout
		buildCmd.Stderr = os.Stderr
		if err := buildCmd.Run(); err != nil {
			return fmt.Errorf("make failed: %w", err)
		}
	} else {
		fmt.Printf("  [%s] No Makefile found, trying go build...\n", repoURL)
		buildCmd := exec.Command("go", "build", "-o", "plugin_binary", ".")
		buildCmd.Dir = tempDir
		buildCmd.Stdout = os.Stdout
		buildCmd.Stderr = os.Stderr
		if err := buildCmd.Run(); err != nil {
			return fmt.Errorf("go build failed: %w", err)
		}
	}

	repoName := strings.TrimSuffix(filepath.Base(repoURL), ".git")
	var binPath string

	// Prefer exact match or "plugin_binary" if we just built it
	if p := filepath.Join(tempDir, repoName); isExecutable(p) {
		binPath = p
	} else if p := filepath.Join(tempDir, "plugin_binary"); isExecutable(p) {
		binPath = p
	} else {
		_ = filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
			if binPath != "" || err != nil || info.IsDir() {
				return nil
			}
			if strings.Contains(path, ".git") {
				return filepath.SkipDir
			}
			if isLikelyBinary(path, info, repoName) {
				binPath = path
			}
			return nil
		})
	}

	if binPath == "" {
		return fmt.Errorf("no executable binary found after build")
	}

	destPath := filepath.Join(pluginsDir, filepath.Base(binPath))
	f, err := os.Open(binPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return writeExecutable(destPath, f)
}

func isLikelyBinary(path string, info os.FileInfo, repoName string) bool {
	if info.Mode()&0111 == 0 {
		return false
	}
	ext := strings.ToLower(filepath.Ext(path))
	ignoredExts := map[string]bool{
		".go": true, ".md": true, ".txt": true, ".yml": true,
		".yaml": true, ".sh": true, ".sum": true, ".mod": true,
		".c": true, ".h": true, ".cpp": true,
	}
	if ignoredExts[ext] {
		return false
	}
	return true
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Mode()&0111 != 0
}

func writeExecutable(path string, r io.Reader) (err error) {
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		closeErr := out.Close()
		if err == nil {
			err = closeErr
		}
	}()

	if _, err = io.Copy(out, r); err != nil {
		return err
	}
	return os.Chmod(path, 0755)
}
