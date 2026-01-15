package navidown

import (
	"strings"
)

// MarkerType identifies the type of navigable element.
type MarkerType int

const (
	MarkerTypeLink MarkerType = iota
	MarkerTypeHeader
)

// MarkerPosition represents the position of a marked element in rendered output.
type MarkerPosition struct {
	Type     MarkerType
	Level    int // Header level 1-6, 0 for links
	LineIdx  int
	StartCol int // visual column (excluding markers and ANSI)
	EndCol   int // visual column (excluding markers and ANSI)
}

// ExtractAllMarkers finds all marker pairs (links and headers) in rendered output.
// Returns positions in document order (top-to-bottom, left-to-right).
func ExtractAllMarkers(renderedLines []string, cleaner LineCleaner) []MarkerPosition {
	var positions []MarkerPosition

	for lineIdx, line := range renderedLines {
		linePositions := extractMarkersFromLine(line, lineIdx, cleaner)
		positions = append(positions, linePositions...)
	}

	return positions
}

// extractMarkersFromLine finds all marker pairs in a single line.
func extractMarkersFromLine(line string, lineIdx int, cleaner LineCleaner) []MarkerPosition {
	var positions []MarkerPosition

	// We need to scan through the line finding markers
	// Track byte position and visual column simultaneously
	runes := []rune(line)

	i := 0
	for i < len(runes) {
		// Check for link start marker (U+200B U+200C)
		if i+1 < len(runes) && runes[i] == '\u200B' && runes[i+1] == '\u200C' {
			pos := extractLinkMarker(runes, i, lineIdx, cleaner)
			if pos != nil {
				positions = append(positions, *pos)
				// Skip past this marker pair to continue searching
				i = findEndOfMarkedContent(runes, i, LinkEndMarker)
				continue
			}
		}

		// Check for header start marker (U+200D followed by U+2060s then U+200D)
		if runes[i] == '\u200D' {
			pos := extractHeaderMarker(runes, i, lineIdx, cleaner)
			if pos != nil {
				positions = append(positions, *pos)
				// Skip past this marker pair
				i = findEndOfMarkedContent(runes, i, HeaderEndMarker)
				continue
			}
		}

		i++
	}

	return positions
}

// extractLinkMarker extracts a link marker position starting at runeIdx.
func extractLinkMarker(runes []rune, runeIdx int, lineIdx int, cleaner LineCleaner) *MarkerPosition {
	// Verify start marker
	if runeIdx+1 >= len(runes) || runes[runeIdx] != '\u200B' || runes[runeIdx+1] != '\u200C' {
		return nil
	}

	// Find the end marker
	endMarkerRunes := []rune(LinkEndMarker)
	endIdx := findMarkerSequence(runes, runeIdx+2, endMarkerRunes)
	if endIdx < 0 {
		return nil
	}

	// Calculate visual columns
	// The content is between (runeIdx+2) and endIdx
	contentStart := runeIdx + 2
	contentEnd := endIdx

	startCol := calculateVisualColumn(runes, contentStart, cleaner)
	endCol := calculateVisualColumn(runes, contentEnd, cleaner)

	return &MarkerPosition{
		Type:     MarkerTypeLink,
		Level:    0,
		LineIdx:  lineIdx,
		StartCol: startCol,
		EndCol:   endCol,
	}
}

// extractHeaderMarker extracts a header marker position starting at runeIdx.
func extractHeaderMarker(runes []rune, runeIdx int, lineIdx int, cleaner LineCleaner) *MarkerPosition {
	// Header start marker format: U+200D + (U+2060 × level) + U+200D
	if runes[runeIdx] != '\u200D' {
		return nil
	}

	// Count U+2060 characters to determine level
	level := 0
	i := runeIdx + 1
	for i < len(runes) && runes[i] == '\u2060' {
		level++
		i++
	}

	// Must have at least one level char and end with U+200D
	if level < 1 || level > 6 || i >= len(runes) || runes[i] != '\u200D' {
		return nil
	}

	// Content starts after the closing U+200D of start marker
	contentStart := i + 1

	// Find the end marker (U+200D U+200C)
	endMarkerRunes := []rune(HeaderEndMarker)
	endIdx := findMarkerSequence(runes, contentStart, endMarkerRunes)
	if endIdx < 0 {
		return nil
	}

	contentEnd := endIdx

	startCol := calculateVisualColumn(runes, contentStart, cleaner)
	endCol := calculateVisualColumn(runes, contentEnd, cleaner)

	return &MarkerPosition{
		Type:     MarkerTypeHeader,
		Level:    level,
		LineIdx:  lineIdx,
		StartCol: startCol,
		EndCol:   endCol,
	}
}

// findMarkerSequence finds the position of a marker sequence in runes starting from startIdx.
// Returns the index where the marker starts, or -1 if not found.
func findMarkerSequence(runes []rune, startIdx int, marker []rune) int {
	for i := startIdx; i <= len(runes)-len(marker); i++ {
		match := true
		for j := 0; j < len(marker); j++ {
			if runes[i+j] != marker[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

// findEndOfMarkedContent finds the position after the end marker.
func findEndOfMarkedContent(runes []rune, startIdx int, endMarker string) int {
	endMarkerRunes := []rune(endMarker)
	idx := findMarkerSequence(runes, startIdx, endMarkerRunes)
	if idx < 0 {
		return len(runes)
	}
	return idx + len(endMarkerRunes)
}

// calculateVisualColumn calculates the visual column at a given rune index.
// This excludes ANSI codes and marker characters from the count.
func calculateVisualColumn(runes []rune, targetIdx int, cleaner LineCleaner) int {
	// Build the string up to targetIdx, clean it, and count visible characters
	var sb strings.Builder
	for i := 0; i < targetIdx && i < len(runes); i++ {
		sb.WriteRune(runes[i])
	}

	// Clean ANSI codes
	cleaned := sb.String()
	if cleaner != nil {
		cleaned = cleaner.Clean(cleaned)
	}

	// Remove marker characters and count remaining
	col := 0
	for _, r := range cleaned {
		if !IsMarkerRune(r) {
			col++
		}
	}

	return col
}
