package navidown

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

var (
	// ErrDirectoryTraversal is returned when a path contains directory traversal attempts.
	ErrDirectoryTraversal = errors.New("directory traversal not allowed")
	// ErrFileNotFound is returned when a file doesn't exist after resolution attempts.
	ErrFileNotFound = errors.New("file not found")
)

// ResolveMarkdownPath resolves a markdown link URL to an absolute file path.
//
// Resolution order:
// 1. If HTTP/HTTPS URL -> return as-is
// 2. Security check for directory traversal
// 3. If absolute path and exists -> return it
// 4. If sourceFilePath provided, try same directory
// 5. Try any extra search roots (in order)
// 6. Return ErrFileNotFound
func ResolveMarkdownPath(linkURL, sourceFilePath string, searchRoots []string) (string, error) {
	if linkURL == "" {
		return "", nil
	}

	if isHTTPURL(linkURL) {
		return linkURL, nil
	}

	if containsDirectoryTraversal(linkURL) {
		return "", ErrDirectoryTraversal
	}

	if filepath.IsAbs(linkURL) {
		if fileExists(linkURL) {
			return linkURL, nil
		}
		return "", ErrFileNotFound
	}

	if sourceFilePath != "" {
		sourceDir := filepath.Dir(sourceFilePath)
		candidate := filepath.Clean(filepath.Join(sourceDir, linkURL))
		if fileExists(candidate) {
			return candidate, nil
		}
	}

	for _, root := range searchRoots {
		if root == "" {
			continue
		}
		candidate := filepath.Clean(filepath.Join(root, linkURL))
		if fileExists(candidate) {
			return candidate, nil
		}
	}

	return "", ErrFileNotFound
}

func isHTTPURL(url string) bool {
	return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")
}

func containsDirectoryTraversal(path string) bool {
	// Security: absolute paths into sensitive system directories are disallowed.
	// This prevents resolving system files like /etc/passwd via absolute link URLs.
	if filepath.IsAbs(path) {
		cleaned := filepath.Clean(path)
		parts := strings.Split(cleaned, string(filepath.Separator))
		// For absolute paths, parts[0] will be "" (leading slash), so the first
		// meaningful segment is parts[1] when present.
		if len(parts) > 1 {
			// block a small set of sensitive system directories.
			// note: do not block /var because temp dirs often live under it (e.g. /var/folders on macOS).
			sensitive := map[string]struct{}{
				"etc":  {},
				"sys":  {},
				"proc": {},
				"root": {},
			}
			if _, ok := sensitive[parts[1]]; ok {
				return true
			}
		}
	}

	if strings.HasPrefix(path, "../") || path == ".." {
		cleaned := filepath.Clean(path)
		parts := strings.Split(cleaned, string(filepath.Separator))

		escapeCount := 0
		for _, part := range parts {
			if part == ".." {
				escapeCount++
			} else {
				break
			}
		}
		if escapeCount > 1 {
			return true
		}

		sensitivePatterns := []string{"etc", "var", "usr", "sys", "proc", "root"}
		for _, part := range parts {
			for _, sensitive := range sensitivePatterns {
				if part == sensitive {
					return true
				}
			}
		}
	}

	if !filepath.IsAbs(path) {
		sensitivePatterns := []string{"etc/", "var/", "usr/", "sys/", "proc/", "root/"}
		for _, pattern := range sensitivePatterns {
			if strings.HasPrefix(path, pattern) || strings.Contains(path, "/"+pattern) {
				return true
			}
		}
	}

	return false
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
