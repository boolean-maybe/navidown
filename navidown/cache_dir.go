package navidown

import (
	"os"
	"path/filepath"
)

// resolveCacheDir determines which directory to use for persistent caching.
// It tries (in order): explicit path, os.UserCacheDir()/navidown/<subdir>, temp dir.
// Returns (persistentDir, tempDir, workDir). workDir is the dir to actually use.
func resolveCacheDir(explicit string, subdir string) (persistentDir, tempDir, workDir string) {
	if explicit != "" {
		if err := os.MkdirAll(explicit, 0700); err == nil {
			return explicit, "", explicit
		}
	}

	if ucd, err := os.UserCacheDir(); err == nil {
		dir := filepath.Join(ucd, "navidown", subdir)
		if err := os.MkdirAll(dir, 0700); err == nil {
			return dir, "", dir
		}
	}

	if td, err := os.MkdirTemp("", "navidown-"+subdir+"-"); err == nil {
		return "", td, td
	}

	return "", "", ""
}
