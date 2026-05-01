package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetPluginsDir(t *testing.T) {
	os.Unsetenv(envPluginsDir)
	if dir := getPluginsDir(); dir != defaultPluginsDir {
		t.Errorf("Expected %s, got %s", defaultPluginsDir, dir)
	}

	customDir := "/custom/plugins"
	os.Setenv(envPluginsDir, customDir)
	if dir := getPluginsDir(); dir != customDir {
		t.Errorf("Expected %s, got %s", customDir, dir)
	}
}

func TestGetFileNameFromURL(t *testing.T) {
	tests := []struct {
		url      string
		expected string
		wantErr  bool
	}{
		{"https://example.com/plugin", "plugin", false},
		{"https://example.com/plugin?v=1.0", "plugin", false},
		{"https://example.com/path/to/my.plugin", "my.plugin", false},
		{"https://example.com/", "", true},
		{"invalid-url", "invalid-url", false}, // url.Parse might not fail on strings without schemes
	}

	for _, tt := range tests {
		got, err := getFileNameFromURL(tt.url)
		if (err != nil) != tt.wantErr {
			t.Errorf("getFileNameFromURL(%s) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			continue
		}
		if got != tt.expected {
			t.Errorf("getFileNameFromURL(%s) = %v, want %v", tt.url, got, tt.expected)
		}
	}
}

type mockFileInfo struct {
	os.FileInfo
	name string
	mode os.FileMode
}

func (m mockFileInfo) Name() string      { return m.name }
func (m mockFileInfo) Mode() os.FileMode { return m.mode }

func TestIsLikelyBinary(t *testing.T) {
	tests := []struct {
		name     string
		mode     os.FileMode
		expected bool
	}{
		{"plugin", 0755, true},
		{"plugin.exe", 0755, true},
		{"main.go", 0755, false},
		{"README.md", 0644, false},
		{"script.sh", 0755, false},
		{"LICENSE", 0644, false},
		{"binary_with.dot", 0755, true},
	}

	for _, tt := range tests {
		info := mockFileInfo{name: tt.name, mode: tt.mode}
		got := isLikelyBinary(filepath.Join("/tmp", tt.name), info, "repo")
		if got != tt.expected {
			t.Errorf("isLikelyBinary(%s, %v) = %v, want %v", tt.name, tt.mode, got, tt.expected)
		}
	}
}
