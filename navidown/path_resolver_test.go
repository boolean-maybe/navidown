package navidown

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestResolveMarkdownPath_HTTPURLs(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{name: "HTTP URL", url: "http://example.com/page.md", expected: "http://example.com/page.md"},
		{name: "HTTPS URL", url: "https://example.com/page.md", expected: "https://example.com/page.md"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ResolveMarkdownPath(tt.url, "", nil)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestResolveMarkdownPath_DirectoryTraversal(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{name: "Parent directory traversal", url: "../../etc/passwd"},
		{name: "Relative path to etc", url: "etc/passwd"},
		{name: "Relative path with /etc", url: "docs/../etc/passwd"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ResolveMarkdownPath(tt.url, "", nil)
			if !errors.Is(err, ErrDirectoryTraversal) {
				t.Errorf("expected ErrDirectoryTraversal, got %v", err)
			}
		})
	}
}

func TestResolveMarkdownPath_AbsolutePath(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(tmpFile, []byte("# Test"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := ResolveMarkdownPath(tmpFile, "", nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != tmpFile {
		t.Errorf("expected %s, got %s", tmpFile, result)
	}
}

func TestResolveMarkdownPath_AbsolutePathNotFound(t *testing.T) {
	nonExistent := filepath.Join(t.TempDir(), "nonexistent.md")
	_, err := ResolveMarkdownPath(nonExistent, "", nil)
	if !errors.Is(err, ErrFileNotFound) {
		t.Errorf("expected ErrFileNotFound, got %v", err)
	}
}

func TestResolveMarkdownPath_SameDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	sourceFile := filepath.Join(tmpDir, "source.md")
	targetFile := filepath.Join(tmpDir, "target.md")

	if err := os.WriteFile(sourceFile, []byte("# Source"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(targetFile, []byte("# Target"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := ResolveMarkdownPath("target.md", sourceFile, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	expectedPath := filepath.Clean(targetFile)
	if result != expectedPath {
		t.Errorf("expected %s, got %s", expectedPath, result)
	}
}

func TestResolveMarkdownPath_SearchRoots(t *testing.T) {
	tmpDir := t.TempDir()
	rootA := filepath.Join(tmpDir, "rootA")
	rootB := filepath.Join(tmpDir, "rootB")
	if err := os.MkdirAll(rootA, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(rootB, 0o755); err != nil {
		t.Fatal(err)
	}

	targetFile := filepath.Join(rootB, "root.md")
	if err := os.WriteFile(targetFile, []byte("# Root"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := ResolveMarkdownPath("root.md", "", []string{rootA, rootB})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	expectedPath := filepath.Clean(targetFile)
	if result != expectedPath {
		t.Errorf("expected %s, got %s", expectedPath, result)
	}
}

func TestResolveMarkdownPath_NotFound(t *testing.T) {
	_, err := ResolveMarkdownPath("nonexistent.md", "", nil)
	if !errors.Is(err, ErrFileNotFound) {
		t.Errorf("expected ErrFileNotFound, got %v", err)
	}
}

func TestResolveMarkdownPath_EmptyURL(t *testing.T) {
	result, err := ResolveMarkdownPath("", "", nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty string, got %s", result)
	}
}

func TestContainsDirectoryTraversal_AbsoluteSensitivePaths(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		want       bool
		skipOnWindows bool
	}{
		{name: "Sensitive /etc path", path: "/etc/passwd", want: true},
		{name: "Non-sensitive /usr path", path: "/usr/bin/env", want: false, skipOnWindows: true},
		{name: "Non-sensitive absolute path", path: "/tmp/test.md", want: false, skipOnWindows: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipOnWindows && runtime.GOOS == "windows" {
				t.Skip("Skipping Unix-specific path test on Windows")
			}
			if got := containsDirectoryTraversal(tt.path); got != tt.want {
				t.Errorf("containsDirectoryTraversal(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
