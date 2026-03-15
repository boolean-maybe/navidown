package navidown

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
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
		limit := 1024
		if len(d) < limit {
			limit = len(d)
		}
		return bytes.Contains(d[:limit], []byte("<svg"))
	}

	return false
}
