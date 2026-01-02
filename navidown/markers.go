package navidown

import "strings"

// Zero-width Unicode characters used for invisible markers
const (
	zws  = "\u200B" // Zero-Width Space
	zwnj = "\u200C" // Zero-Width Non-Joiner
	zwj  = "\u200D" // Zero-Width Joiner
	wj   = "\u2060" // Word Joiner
)

// Link markers - unique sequence to identify link text boundaries
const (
	LinkStartMarker = zws + zwnj // U+200B U+200C
	LinkEndMarker   = zwnj + zws // U+200C U+200B
)

// Header markers - level is encoded by repeating Word Joiner (U+2060)
// Start: ZWJ + (WJ × level) + ZWJ
// End: ZWJ + ZWNJ (same for all levels)
const (
	headerMarkerPrefix = zwj        // U+200D
	headerLevelChar    = wj         // U+2060
	HeaderEndMarker    = zwj + zwnj // U+200D U+200C
)

// HeaderStartMarker generates a start marker for a heading with encoded level.
// Level must be 1-6.
func HeaderStartMarker(level int) string {
	if level < 1 {
		level = 1
	}
	if level > 6 {
		level = 6
	}
	return headerMarkerPrefix + strings.Repeat(headerLevelChar, level) + headerMarkerPrefix
}

// DecodeHeaderLevel extracts the heading level from a header start marker.
// Returns 0 if the marker is not a valid header start marker.
func DecodeHeaderLevel(marker string) int {
	if !strings.HasPrefix(marker, headerMarkerPrefix) {
		return 0
	}
	if !strings.HasSuffix(marker, headerMarkerPrefix) {
		return 0
	}

	// Count the Word Joiner characters between the ZWJ prefix and suffix
	inner := marker[len(headerMarkerPrefix) : len(marker)-len(headerMarkerPrefix)]
	level := strings.Count(inner, headerLevelChar)

	if level < 1 || level > 6 {
		return 0
	}
	return level
}

// IsMarkerRune returns true if the rune is one of our marker characters.
func IsMarkerRune(r rune) bool {
	return r == '\u200B' || r == '\u200C' || r == '\u200D' || r == '\u2060'
}

// StripMarkers removes all marker characters from a string.
func StripMarkers(s string) string {
	var result strings.Builder
	result.Grow(len(s))
	for _, r := range s {
		if !IsMarkerRune(r) {
			result.WriteRune(r)
		}
	}
	return result.String()
}
