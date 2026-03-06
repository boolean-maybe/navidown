package navidown

import (
	"fmt"
	"strings"
)

// Image placeholder tokens used in rendered output.
// These bracket an image reference that will be replaced with Kitty placeholders
// during post-processing, or left as alt-text when image support is disabled.
const (
	ImageTokenStart = "\uFFF0"
	ImageTokenEnd   = "\uFFF1"
)

// ImagePlaceholder represents a resolved image ready for placeholder generation.
type ImagePlaceholder struct {
	ImageID uint32 // Unique ID for this image (1-based, encoded in fg color)
	Cols    int    // Width in terminal columns
	Rows    int    // Height in terminal rows
	URL     string // Original URL for reference
}

// imageFieldSep separates URL from alt text within the token.
// Using a null byte since it can't appear in URLs or markdown text.
const imageFieldSep = "\x00"

// FormatImageToken creates the deferred placeholder token that glamour emits.
// Format: \uFFF0IMG:<url>\x00<alt>\uFFF1
func FormatImageToken(url, altText string) string {
	return ImageTokenStart + "IMG:" + url + imageFieldSep + altText + ImageTokenEnd
}

// ParseImageToken extracts URL and alt text from a placeholder token.
// Returns url, altText, ok.
func ParseImageToken(token string) (string, string, bool) {
	if !strings.HasPrefix(token, ImageTokenStart+"IMG:") {
		return "", "", false
	}
	if !strings.HasSuffix(token, ImageTokenEnd) {
		return "", "", false
	}

	// Strip delimiters: \uFFF0IMG: ... \uFFF1
	inner := token[len(ImageTokenStart+"IMG:") : len(token)-len(ImageTokenEnd)]

	// Split on null byte separator
	url, altText, found := strings.Cut(inner, imageFieldSep)
	if !found {
		return inner, "", true
	}
	return url, altText, true
}

// ContainsImageToken returns true if the line contains an image placeholder token.
func ContainsImageToken(line string) bool {
	return strings.Contains(line, ImageTokenStart+"IMG:")
}

// ImageIDToColor encodes a 1-based image ID into an RGB foreground color string.
// Kitty uses the foreground color to identify which image to render.
// Format: #RRGGBB where the 24-bit value encodes the image ID.
func ImageIDToColor(id uint32) string {
	if id == 0 {
		id = 1 // 0 is not valid for Kitty image IDs
	}
	r := (id >> 16) & 0xFF
	g := (id >> 8) & 0xFF
	b := id & 0xFF
	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

// IsImageMarkerRune returns true if the rune is one of the image token delimiters.
func IsImageMarkerRune(r rune) bool {
	return r == '\uFFF0' || r == '\uFFF1'
}

// ImagePostProcessor replaces image placeholder tokens in rendered output
// with actual displayable content (e.g., Kitty Unicode placeholder lines).
type ImagePostProcessor interface {
	// ProcessImageTokens scans rendered lines for image tokens and replaces
	// them with displayable content. Returns the modified lines.
	// sourceFilePath is provided for relative URL resolution.
	// maxCols is the available viewport width.
	ProcessImageTokens(lines []string, sourceFilePath string, maxCols int) []string
}

// FallbackImageProcessor replaces image tokens with simple alt text.
// Used when Kitty graphics protocol is not available.
type FallbackImageProcessor struct{}

func (p *FallbackImageProcessor) ProcessImageTokens(lines []string, _ string, _ int) []string {
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		if !ContainsImageToken(line) {
			result = append(result, line)
			continue
		}
		// Replace tokens with alt text
		replaced := replaceImageTokensInLine(line, func(url, alt string) string {
			if alt != "" {
				return "[image: " + alt + "]"
			}
			return "[image]"
		})
		result = append(result, replaced)
	}
	return result
}

// replaceImageTokensInLine replaces all image tokens in a single line using the
// provided replacement function. The function receives (url, altText) and returns
// the replacement string.
func replaceImageTokensInLine(line string, replacer func(url, alt string) string) string {
	var result strings.Builder
	remaining := line
	for {
		startIdx := strings.Index(remaining, ImageTokenStart+"IMG:")
		if startIdx < 0 {
			result.WriteString(remaining)
			break
		}
		result.WriteString(remaining[:startIdx])

		endIdx := strings.Index(remaining[startIdx:], ImageTokenEnd)
		if endIdx < 0 {
			result.WriteString(remaining[startIdx:])
			break
		}
		endIdx += startIdx + len(ImageTokenEnd)

		token := remaining[startIdx:endIdx]
		url, alt, ok := ParseImageToken(token)
		if ok {
			result.WriteString(replacer(url, alt))
		} else {
			result.WriteString(token)
		}
		remaining = remaining[endIdx:]
	}
	return result.String()
}
