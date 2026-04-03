package navidown

import (
	"errors"
	"net/url"
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
// 1. If linkURL is HTTP/HTTPS -> return as-is
// 2. If sourceFilePath is HTTP/HTTPS -> resolve linkURL as URL reference
// 3. Security check for local directory traversal
// 4. If local absolute path and exists -> return it
// 5. If local sourceFilePath provided, try same directory
// 6. Try any extra local search roots (in order)
// 7. Return ErrFileNotFound
//
// Note: in HTTP source mode, resolution is URL-only and does not fall back to
// local path checks or search roots.
func ResolveMarkdownPath(linkURL, sourceFilePath string, searchRoots []string) (string, error) {
	if linkURL == "" {
		return "", nil
	}

	if isHTTPURL(linkURL) {
		return linkURL, nil
	}

	// resolve relative links against HTTP source file path;
	// commits to remote resolution — no local fallback
	if sourceFilePath != "" && looksLikeHTTPURL(sourceFilePath) {
		return resolveAgainstHTTPSource(sourceFilePath, linkURL)
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

// looksLikeHTTPURL is a case-insensitive prefix check for http(s) URLs.
// Separate from isHTTPURL to avoid changing its callers.
func looksLikeHTTPURL(s string) bool {
	lower := strings.ToLower(s)
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://")
}

// resolveAgainstHTTPSource resolves a relative link against an HTTP source URL
// using RFC 3986 reference resolution. Returns only http/https results with
// non-empty host. Returns ErrFileNotFound for invalid inputs to preserve the
// existing sentinel error contract.
func resolveAgainstHTTPSource(sourceURL, linkURL string) (string, error) {
	base, err := url.Parse(sourceURL)
	if err != nil || base.Host == "" {
		return "", ErrFileNotFound
	}
	ref, err := url.Parse(linkURL)
	if err != nil {
		return "", ErrFileNotFound
	}
	resolved := base.ResolveReference(ref)
	if resolved.Host == "" || !isHTTPScheme(resolved.Scheme) {
		return "", ErrFileNotFound
	}
	return resolved.String(), nil
}

// isHTTPScheme checks if a scheme is http or https (case-insensitive).
func isHTTPScheme(scheme string) bool {
	return strings.EqualFold(scheme, "http") || strings.EqualFold(scheme, "https")
}
