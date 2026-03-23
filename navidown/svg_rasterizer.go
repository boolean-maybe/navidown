package navidown

import (
	"bytes"
	"crypto/sha256"
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
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

// parseSVGDimensions extracts intrinsic width and height from SVG data.
// It checks the root <svg> element's width/height attributes first,
// falling back to viewBox dimensions. Returns (0, 0, false) if no
// intrinsic dimensions can be determined.
func parseSVGDimensions(data []byte) (width, height float64, ok bool) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.Strict = false
	decoder.AutoClose = xml.HTMLAutoClose

	for {
		tok, err := decoder.Token()
		if err != nil {
			return 0, 0, false
		}
		se, isSE := tok.(xml.StartElement)
		if !isSE {
			continue
		}
		if se.Name.Local != "svg" {
			continue
		}

		var rawW, rawH, viewBox string
		for _, attr := range se.Attr {
			switch attr.Name.Local {
			case "width":
				rawW = attr.Value
			case "height":
				rawH = attr.Value
			case "viewBox":
				viewBox = attr.Value
			}
		}

		w, wOK := parseSVGLength(rawW)
		h, hOK := parseSVGLength(rawH)
		if wOK && hOK && w > 0 && h > 0 {
			return w, h, true
		}

		// fall back to viewBox
		if vw, vh, vOK := parseViewBox(viewBox); vOK {
			return vw, vh, true
		}
		return 0, 0, false
	}
}

// parseSVGLength parses a CSS length value (e.g. "90", "90px", "72pt").
// Returns the value in pixels and true, or (0, false) for unparseable
// or viewport-relative values (%, em, rem).
func parseSVGLength(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}

	// separate numeric prefix from unit suffix
	i := 0
	for i < len(s) && (s[i] == '.' || s[i] == '-' || s[i] == '+' || (s[i] >= '0' && s[i] <= '9')) {
		i++
	}
	if i == 0 {
		return 0, false
	}

	num, err := strconv.ParseFloat(s[:i], 64)
	if err != nil {
		return 0, false
	}

	unit := strings.TrimSpace(s[i:])
	switch unit {
	case "", "px":
		return num, true
	case "pt":
		return num * 96.0 / 72.0, true
	case "in":
		return num * 96.0, true
	case "cm":
		return num * 96.0 / 2.54, true
	case "mm":
		return num * 96.0 / 25.4, true
	default:
		// %, em, rem, ex, etc. — viewport-relative, skip
		return 0, false
	}
}

// parseViewBox extracts width and height from a viewBox attribute value
// formatted as "minX minY width height".
func parseViewBox(s string) (width, height float64, ok bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, 0, false
	}
	parts := strings.Fields(s)
	if len(parts) != 4 {
		return 0, 0, false
	}
	w, err := strconv.ParseFloat(parts[2], 64)
	if err != nil || w <= 0 {
		return 0, 0, false
	}
	h, err := strconv.ParseFloat(parts[3], 64)
	if err != nil || h <= 0 {
		return 0, 0, false
	}
	return w, h, true
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
