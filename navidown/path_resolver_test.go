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
	if err := os.MkdirAll(rootA, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(rootB, 0o750); err != nil {
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

func TestResolveMarkdownPath_MultipleParentSegments(t *testing.T) {
	// Build: tmpDir/project/.doc/tiki/tiki-test.md referencing ../../res/image.png
	// Target: tmpDir/project/res/image.png
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "project")
	docDir := filepath.Join(projectDir, ".doc", "tiki")
	resDir := filepath.Join(projectDir, "res")

	if err := os.MkdirAll(docDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(resDir, 0o750); err != nil {
		t.Fatal(err)
	}

	sourceFile := filepath.Join(docDir, "tiki-test.md")
	if err := os.WriteFile(sourceFile, []byte("# Test"), 0o644); err != nil {
		t.Fatal(err)
	}
	imageFile := filepath.Join(resDir, "image.png")
	if err := os.WriteFile(imageFile, []byte("PNG"), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{name: "Two parent segments", url: "../../res/image.png", expected: imageFile},
		{name: "One parent segment", url: "../tiki/tiki-test.md", expected: sourceFile},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ResolveMarkdownPath(tt.url, sourceFile, nil)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestResolveMarkdownPath_DeepTraversalToSensitiveDir(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{name: "Deep traversal to etc", url: "../../../etc/passwd"},
		{name: "Two segments to etc", url: "../../etc/passwd"},
		{name: "One segment to proc", url: "../proc/cpuinfo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ResolveMarkdownPath(tt.url, "", nil)
			if !errors.Is(err, ErrDirectoryTraversal) {
				t.Errorf("expected ErrDirectoryTraversal for %q, got %v", tt.url, err)
			}
		})
	}
}

func TestResolveMarkdownPath_HTTPSourceFilePath(t *testing.T) {
	tests := []struct {
		name           string
		linkURL        string
		sourceFilePath string
		expected       string
	}{
		{
			name:           "relative path in subdirectory",
			linkURL:        "svg/image.svg",
			sourceFilePath: "https://host/dir/file.md",
			expected:       "https://host/dir/svg/image.svg",
		},
		{
			name:           "dot-relative path",
			linkURL:        "./img.png",
			sourceFilePath: "https://host/dir/file.md",
			expected:       "https://host/dir/img.png",
		},
		{
			name:           "parent directory traversal",
			linkURL:        "../other/img.png",
			sourceFilePath: "https://host/dir/sub/file.md",
			expected:       "https://host/dir/other/img.png",
		},
		{
			name:           "simple filename at root",
			linkURL:        "img.png",
			sourceFilePath: "https://host/file.md",
			expected:       "https://host/img.png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ResolveMarkdownPath(tt.linkURL, tt.sourceFilePath, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestResolveMarkdownPath_RootRelativeFromHTTPSource(t *testing.T) {
	result, err := ResolveMarkdownPath("/absolute/img.png", "https://host/dir/file.md", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "https://host/absolute/img.png"
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestResolveMarkdownPath_SchemeRelativeFromHTTPSource(t *testing.T) {
	tests := []struct {
		name           string
		linkURL        string
		sourceFilePath string
		expected       string
	}{
		{
			name:           "cross-host scheme-relative",
			linkURL:        "//other-host/img.png",
			sourceFilePath: "https://host/dir/file.md",
			expected:       "https://other-host/img.png",
		},
		{
			name:           "CDN scheme-relative",
			linkURL:        "//cdn.example/assets/logo.svg",
			sourceFilePath: "https://host/dir/file.md",
			expected:       "https://cdn.example/assets/logo.svg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ResolveMarkdownPath(tt.linkURL, tt.sourceFilePath, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestResolveMarkdownPath_HTTPSourceRejectsInvalidInputs(t *testing.T) {
	tests := []struct {
		name           string
		linkURL        string
		sourceFilePath string
	}{
		{
			name:           "file scheme in link",
			linkURL:        "file:///tmp/img.png",
			sourceFilePath: "https://host/dir/file.md",
		},
		{
			name:           "data scheme in link",
			linkURL:        "data:text/plain,hello",
			sourceFilePath: "https://host/dir/file.md",
		},
		{
			name:           "empty host in source",
			linkURL:        "img.png",
			sourceFilePath: "https://",
		},
		{
			name:           "empty host with path in source",
			linkURL:        "img.png",
			sourceFilePath: "https:///dir/file.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ResolveMarkdownPath(tt.linkURL, tt.sourceFilePath, nil)
			if !errors.Is(err, ErrFileNotFound) {
				t.Errorf("expected ErrFileNotFound, got %v", err)
			}
		})
	}
}

func TestResolveMarkdownPath_HTTPSourceWithTraversalLikePaths(t *testing.T) {
	tests := []struct {
		name           string
		linkURL        string
		sourceFilePath string
		expected       string
	}{
		{
			name:           "traversal to etc in remote context",
			linkURL:        "../etc/config.svg",
			sourceFilePath: "https://host/dir/sub/file.md",
			expected:       "https://host/dir/etc/config.svg",
		},
		{
			name:           "double traversal in remote context",
			linkURL:        "../../img.png",
			sourceFilePath: "https://host/a/b/c/file.md",
			expected:       "https://host/a/img.png",
		},
		{
			name:           "absolute sensitive path in remote context",
			linkURL:        "/etc/passwd",
			sourceFilePath: "https://host/dir/file.md",
			expected:       "https://host/etc/passwd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ResolveMarkdownPath(tt.linkURL, tt.sourceFilePath, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestResolveMarkdownPath_HTTPSourceWithQueryAndFragment(t *testing.T) {
	tests := []struct {
		name           string
		linkURL        string
		sourceFilePath string
		expected       string
	}{
		{
			name:           "source with query parameter",
			linkURL:        "img.png",
			sourceFilePath: "https://host/dir/file.md?raw=1",
			expected:       "https://host/dir/img.png",
		},
		{
			name:           "source with fragment",
			linkURL:        "img.png",
			sourceFilePath: "https://host/dir/file.md#section",
			expected:       "https://host/dir/img.png",
		},
		{
			name:           "link with query parameter",
			linkURL:        "img.png?v=2",
			sourceFilePath: "https://host/dir/file.md",
			expected:       "https://host/dir/img.png?v=2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ResolveMarkdownPath(tt.linkURL, tt.sourceFilePath, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestResolveMarkdownPath_HTTPSourceNoLocalFallback(t *testing.T) {
	// even with a valid local searchRoot, HTTP source mode must not fall back
	tmpDir := t.TempDir()
	targetFile := filepath.Join(tmpDir, "img.png")
	if err := os.WriteFile(targetFile, []byte("PNG"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := ResolveMarkdownPath("img.png", "https:///invalid", []string{tmpDir})
	if !errors.Is(err, ErrFileNotFound) {
		t.Errorf("expected ErrFileNotFound (no local fallback), got %v", err)
	}
}

func TestResolveMarkdownPath_LocalNotAffectedByHTTPChanges(t *testing.T) {
	// local traversal is still blocked
	_, err := ResolveMarkdownPath("../etc/passwd", "", nil)
	if !errors.Is(err, ErrDirectoryTraversal) {
		t.Errorf("expected ErrDirectoryTraversal for local traversal, got %v", err)
	}

	// local source resolution still works
	tmpDir := t.TempDir()
	sourceFile := filepath.Join(tmpDir, "source.md")
	targetFile := filepath.Join(tmpDir, "img.png")
	if err := os.WriteFile(sourceFile, []byte("# Source"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(targetFile, []byte("PNG"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := ResolveMarkdownPath("img.png", sourceFile, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != filepath.Clean(targetFile) {
		t.Errorf("expected %s, got %s", filepath.Clean(targetFile), result)
	}
}

func TestContainsDirectoryTraversal_AbsoluteSensitivePaths(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		want          bool
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
