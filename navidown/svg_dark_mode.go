package navidown

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// fillStrokePattern matches fill="..." and stroke="..." attributes in SVG.
var fillStrokePattern = regexp.MustCompile(`(fill|stroke)="([^"]*)"`)

// preprocessSVGForDarkMode rewrites dark gray fill and stroke colors to
// light equivalents so SVG text and lines are visible on dark backgrounds.
// Colors with significant hue (blues, greens, etc.) are left unchanged
// since they typically appear on colored boxes that provide contrast.
func preprocessSVGForDarkMode(data []byte) []byte {
	s := string(data)
	result := fillStrokePattern.ReplaceAllStringFunc(s, func(match string) string {
		parts := fillStrokePattern.FindStringSubmatch(match)
		if len(parts) != 3 {
			return match
		}
		attr, color := parts[1], parts[2]

		remapped, ok := remapDarkColor(color)
		if !ok {
			return match
		}
		return fmt.Sprintf(`%s="%s"`, attr, remapped)
	})
	return []byte(result)
}

// remapDarkColor returns a lightened color and true if the input is a dark
// gray that would be invisible on a dark background. Returns ("", false)
// if the color should be left unchanged.
func remapDarkColor(color string) (string, bool) {
	color = strings.TrimSpace(color)

	// handle named color
	if strings.EqualFold(color, "black") {
		return "#e0e0e0", true
	}

	if !strings.HasPrefix(color, "#") {
		return "", false
	}

	r, g, b, ok := parseHexColor(color)
	if !ok {
		return "", false
	}

	if isDarkGray(r, g, b) {
		return invertGray(r, g, b), true
	}
	return "", false
}

// parseHexColor parses #RGB or #RRGGBB to r, g, b values (0-255).
func parseHexColor(s string) (r, g, b uint8, ok bool) {
	s = strings.TrimPrefix(s, "#")
	switch len(s) {
	case 3:
		rv, err1 := strconv.ParseUint(string(s[0])+string(s[0]), 16, 8)
		gv, err2 := strconv.ParseUint(string(s[1])+string(s[1]), 16, 8)
		bv, err3 := strconv.ParseUint(string(s[2])+string(s[2]), 16, 8)
		if err1 != nil || err2 != nil || err3 != nil {
			return 0, 0, 0, false
		}
		return uint8(rv), uint8(gv), uint8(bv), true
	case 6:
		rv, err1 := strconv.ParseUint(s[0:2], 16, 8)
		gv, err2 := strconv.ParseUint(s[2:4], 16, 8)
		bv, err3 := strconv.ParseUint(s[4:6], 16, 8)
		if err1 != nil || err2 != nil || err3 != nil {
			return 0, 0, 0, false
		}
		return uint8(rv), uint8(gv), uint8(bv), true
	default:
		return 0, 0, 0, false
	}
}

// isDarkGray returns true if the color is an achromatic gray (low saturation)
// and dark enough to be invisible on a dark background.
// Gray test: max(R,G,B) - min(R,G,B) < 30
// Dark test: average channel < 128
func isDarkGray(r, g, b uint8) bool {
	maxC := max(r, max(g, b))
	minC := min(r, min(g, b))
	if int(maxC)-int(minC) >= 30 {
		return false // has significant hue
	}
	avg := (int(r) + int(g) + int(b)) / 3
	return avg < 128
}

// invertGray returns the luminance-inverted hex color string.
// e.g. (0x33, 0x33, 0x33) → "#cccccc"
func invertGray(r, g, b uint8) string {
	return fmt.Sprintf("#%02x%02x%02x", 255-r, 255-g, 255-b)
}
