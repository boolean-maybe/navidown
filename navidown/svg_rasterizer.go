package navidown

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
)

// SVGRasterizer converts SVG data into raster PNG bytes.
type SVGRasterizer interface {
	Rasterize(svgData []byte, targetWidth int) ([]byte, error)
}

// ResvgRasterizer rasterizes SVG via the `resvg` CLI tool.
type ResvgRasterizer struct{}

func (r *ResvgRasterizer) Rasterize(svgData []byte, targetWidth int) ([]byte, error) {
	resvgPath, err := exec.LookPath("resvg")
	if err != nil {
		return nil, fmt.Errorf("resvg not found in PATH: %w", err)
	}

	cmd := exec.Command(resvgPath, "-w", strconv.Itoa(targetWidth), "-", "-c") // #nosec G204 -- resvgPath from LookPath("resvg")
	cmd.Stdin = bytes.NewReader(svgData)

	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("resvg failed: %s", exitErr.Stderr)
		}
		return nil, fmt.Errorf("resvg: %w", err)
	}

	return out, nil
}

// CachingSVGRasterizer wraps an SVGRasterizer with a persistent disk cache
// (similar to MermaidRenderer). Cache key is SHA256(svgData + \x00 + targetWidth).
type CachingSVGRasterizer struct {
	inner         SVGRasterizer
	cache         sync.Map // hex key -> []byte (PNG data)
	persistentDir string
	tempDir       string
	workDir       string
}

// NewCachingSVGRasterizer creates a caching wrapper around the given rasterizer.
// cacheDir is an explicit cache directory; "" uses os.UserCacheDir()/navidown/svg.
// Returns nil if no usable cache directory can be resolved.
func NewCachingSVGRasterizer(inner SVGRasterizer, cacheDir string) *CachingSVGRasterizer {
	persistentDir, tempDir, workDir := resolveCacheDir(cacheDir, "svg")
	if workDir == "" {
		return nil
	}
	return &CachingSVGRasterizer{
		inner:         inner,
		persistentDir: persistentDir,
		tempDir:       tempDir,
		workDir:       workDir,
	}
}

func svgCacheKey(svgData []byte, targetWidth int) string {
	h := sha256.New()
	h.Write(svgData)
	h.Write([]byte{0})
	_, _ = fmt.Fprintf(h, "%d", targetWidth)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func (c *CachingSVGRasterizer) Rasterize(svgData []byte, targetWidth int) ([]byte, error) {
	key := svgCacheKey(svgData, targetWidth)

	// in-memory cache
	if cached, ok := c.cache.Load(key); ok {
		data, _ := cached.([]byte)
		return data, nil
	}

	// disk cache
	diskPath := filepath.Join(c.workDir, key+".png")
	if data, err := os.ReadFile(diskPath); err == nil {
		c.cache.Store(key, data)
		return data, nil
	}

	// run rasterizer
	pngData, err := c.inner.Rasterize(svgData, targetWidth)
	if err != nil {
		return nil, err
	}

	// write to disk (best effort)
	_ = os.WriteFile(diskPath, pngData, 0600)

	c.cache.Store(key, pngData)
	return pngData, nil
}

// Close removes the temp directory if one was used as fallback.
// Persistent cache directories are preserved.
func (c *CachingSVGRasterizer) Close() {
	if c.tempDir != "" {
		_ = os.RemoveAll(c.tempDir)
	}
}

// isSVGData returns true if data looks like an SVG document.
// Checks for <svg prefix or <?xml prefix followed by <svg within the first 1KB.
func isSVGData(data []byte) bool {
	// strip UTF-8 BOM if present, then trim whitespace
	d := data
	if len(d) >= 3 && d[0] == 0xEF && d[1] == 0xBB && d[2] == 0xBF {
		d = d[3:]
	}
	d = bytes.TrimLeft(d, " \t\r\n")

	if bytes.HasPrefix(d, []byte("<svg")) {
		return true
	}

	if bytes.HasPrefix(d, []byte("<?xml")) {
		limit := min(1024, len(d))
		return bytes.Contains(d[:limit], []byte("<svg"))
	}

	return false
}
