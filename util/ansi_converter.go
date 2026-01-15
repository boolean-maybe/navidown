package util

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Package-level compiled regex for ANSI SGR sequences (avoids recompilation per call).
var ansiSGRPattern = regexp.MustCompile(`\x1b\[([0-9;]*)m`)

// AnsiConverter converts ANSI escape sequences to tview color tags.
type AnsiConverter struct {
	enabled bool
}

// NewAnsiConverter creates a new ANSI converter.
// enabled: if false, returns text unchanged.
func NewAnsiConverter(enabled bool) *AnsiConverter {
	return &AnsiConverter{
		enabled: enabled,
	}
}

// Convert translates ANSI escape sequences to tview color tags.
// Properly handles foreground, background, and bold attributes.
func (c *AnsiConverter) Convert(text string) string {
	if !c.enabled {
		return text
	}

	result := strings.Builder{}
	lastIndex := 0

	var fgColor, bgColor string
	bold := false

	matches := ansiSGRPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		result.WriteString(text[lastIndex:match[0]])

		params := text[match[2]:match[3]]

		newFg, newBg, newBold := parseSGR(params, fgColor, bgColor, bold)
		if newFg != fgColor || newBg != bgColor || newBold != bold {
			fgColor = newFg
			bgColor = newBg
			bold = newBold
			result.WriteString(formatTviewTag(fgColor, bgColor, bold))
		}

		lastIndex = match[1]
	}

	result.WriteString(text[lastIndex:])
	return result.String()
}

func parseSGR(params string, currentFg, currentBg string, currentBold bool) (fg, bg string, bold bool) {
	fg = currentFg
	bg = currentBg
	bold = currentBold

	// Per ANSI SGR, an empty parameter list (ESC[m) is equivalent to "0" (reset).
	if params == "" {
		params = "0"
	}

	parts := strings.Split(params, ";")

	for i := 0; i < len(parts); i++ {
		code, err := strconv.Atoi(parts[i])
		if err != nil {
			continue
		}

		switch code {
		case 0:
			fg = ""
			bg = ""
			bold = false
		case 1:
			bold = true
		case 22:
			bold = false
		case 38:
			if i+2 < len(parts) && parts[i+1] == "5" {
				if colorCode, err := strconv.Atoi(parts[i+2]); err == nil {
					fg = Ansi256ToHex(colorCode)
					i += 2
				}
			} else if i+4 < len(parts) && parts[i+1] == "2" {
				r, _ := strconv.Atoi(parts[i+2])
				g, _ := strconv.Atoi(parts[i+3])
				b, _ := strconv.Atoi(parts[i+4])
				fg = fmt.Sprintf("#%02x%02x%02x", r, g, b)
				i += 4
			}
		case 48:
			if i+2 < len(parts) && parts[i+1] == "5" {
				if colorCode, err := strconv.Atoi(parts[i+2]); err == nil {
					bg = Ansi256ToHex(colorCode)
					i += 2
				}
			} else if i+4 < len(parts) && parts[i+1] == "2" {
				r, _ := strconv.Atoi(parts[i+2])
				g, _ := strconv.Atoi(parts[i+3])
				b, _ := strconv.Atoi(parts[i+4])
				bg = fmt.Sprintf("#%02x%02x%02x", r, g, b)
				i += 4
			}
		case 39:
			fg = ""
		case 49:
			bg = ""
		}
	}

	return fg, bg, bold
}

func formatTviewTag(fg, bg string, bold bool) string {
	if fg == "" {
		fg = "-"
	}
	if bg == "" {
		bg = "-"
	}

	attr := "-"
	if bold {
		attr = "b"
	}

	return fmt.Sprintf("[%s:%s:%s]", fg, bg, attr)
}

// Ansi256ToHex converts ANSI 256 color code to hex color.
func Ansi256ToHex(code int) string {
	r, g, b := Ansi256ToRGB(code)
	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

// Ansi256ToRGB converts ANSI 256 color code to RGB values.
func Ansi256ToRGB(code int) (r, g, b int) {
	if code < 16 {
		standardColors := [][]int{
			{0, 0, 0}, {128, 0, 0}, {0, 128, 0}, {128, 128, 0},
			{0, 0, 128}, {128, 0, 128}, {0, 128, 128}, {192, 192, 192},
			{128, 128, 128}, {255, 0, 0}, {0, 255, 0}, {255, 255, 0},
			{0, 0, 255}, {255, 0, 255}, {0, 255, 255}, {255, 255, 255},
		}
		if code < len(standardColors) {
			return standardColors[code][0], standardColors[code][1], standardColors[code][2]
		}
	} else if code >= 16 && code <= 231 {
		code -= 16
		b := code % 6
		g := (code / 6) % 6
		r := code / 36
		return r * 51, g * 51, b * 51
	} else if code >= 232 && code <= 255 {
		gray := 8 + (code-232)*10
		return gray, gray, gray
	}
	return 0, 0, 0
}
