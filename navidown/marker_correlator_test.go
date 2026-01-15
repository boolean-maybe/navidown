package navidown

import (
	"strings"
	"testing"
)

func TestMarkerExtractor_ExtractsLinkMarkers(t *testing.T) {
	// Simulate rendered line with link markers
	line := "Some text " + LinkStartMarker + "Click here" + LinkEndMarker + " more text"
	lines := []string{line}

	positions := ExtractAllMarkers(lines, nil)

	if len(positions) != 1 {
		t.Fatalf("expected 1 marker position, got %d", len(positions))
	}

	pos := positions[0]
	if pos.Type != MarkerTypeLink {
		t.Errorf("expected MarkerTypeLink, got %v", pos.Type)
	}
	if pos.LineIdx != 0 {
		t.Errorf("expected LineIdx 0, got %d", pos.LineIdx)
	}
	// "Some text " = 10 chars, then link text starts
	if pos.StartCol != 10 {
		t.Errorf("expected StartCol 10, got %d", pos.StartCol)
	}
	// "Click here" = 10 chars
	if pos.EndCol != 20 {
		t.Errorf("expected EndCol 20, got %d", pos.EndCol)
	}
}

func TestMarkerExtractor_ExtractsHeaderMarkers(t *testing.T) {
	// Test different header levels
	for level := 1; level <= 6; level++ {
		line := HeaderStartMarker(level) + "Header Text" + HeaderEndMarker
		lines := []string{line}

		positions := ExtractAllMarkers(lines, nil)

		if len(positions) != 1 {
			t.Fatalf("level %d: expected 1 marker position, got %d", level, len(positions))
		}

		pos := positions[0]
		if pos.Type != MarkerTypeHeader {
			t.Errorf("level %d: expected MarkerTypeHeader, got %v", level, pos.Type)
		}
		if pos.Level != level {
			t.Errorf("expected Level %d, got %d", level, pos.Level)
		}
	}
}

func TestMarkerExtractor_MultipleMarkersOnOneLine(t *testing.T) {
	// Line with two links
	line := LinkStartMarker + "Link1" + LinkEndMarker + " and " + LinkStartMarker + "Link2" + LinkEndMarker
	lines := []string{line}

	positions := ExtractAllMarkers(lines, nil)

	if len(positions) != 2 {
		t.Fatalf("expected 2 marker positions, got %d", len(positions))
	}

	if positions[0].StartCol != 0 || positions[0].EndCol != 5 {
		t.Errorf("first link position wrong: start=%d end=%d", positions[0].StartCol, positions[0].EndCol)
	}
	// " and " = 5 chars, so second link starts at 5+5=10
	if positions[1].StartCol != 10 || positions[1].EndCol != 15 {
		t.Errorf("second link position wrong: start=%d end=%d", positions[1].StartCol, positions[1].EndCol)
	}
}

func TestMarkerExtractor_MixedHeadersAndLinks(t *testing.T) {
	lines := []string{
		HeaderStartMarker(1) + "Title" + HeaderEndMarker,
		"Some text with " + LinkStartMarker + "a link" + LinkEndMarker,
		HeaderStartMarker(2) + "Subtitle" + HeaderEndMarker,
	}

	positions := ExtractAllMarkers(lines, nil)

	if len(positions) != 3 {
		t.Fatalf("expected 3 marker positions, got %d", len(positions))
	}

	// First: H1 header
	if positions[0].Type != MarkerTypeHeader || positions[0].Level != 1 || positions[0].LineIdx != 0 {
		t.Errorf("first position wrong: %+v", positions[0])
	}

	// Second: Link
	if positions[1].Type != MarkerTypeLink || positions[1].LineIdx != 1 {
		t.Errorf("second position wrong: %+v", positions[1])
	}

	// Third: H2 header
	if positions[2].Type != MarkerTypeHeader || positions[2].Level != 2 || positions[2].LineIdx != 2 {
		t.Errorf("third position wrong: %+v", positions[2])
	}
}

func TestMarkerCorrelator_CorrelatesLinksInOrder(t *testing.T) {
	mc := NewMarkerCorrelator()
	mc.Reset()

	lines := []string{
		LinkStartMarker + "First" + LinkEndMarker + " text " + LinkStartMarker + "Second" + LinkEndMarker,
		"Another line with " + LinkStartMarker + "Third" + LinkEndMarker,
	}

	elem1 := &NavElement{Type: NavElementURL, Text: "First"}
	elem2 := &NavElement{Type: NavElementURL, Text: "Second"}
	elem3 := &NavElement{Type: NavElementURL, Text: "Third"}

	// First correlation
	lineIdx, startCol, endCol, found := mc.CorrelatePosition(elem1, lines, nil)
	if !found || lineIdx != 0 || startCol != 0 || endCol != 5 {
		t.Errorf("first link wrong: found=%v line=%d start=%d end=%d", found, lineIdx, startCol, endCol)
	}

	// Second correlation
	lineIdx, startCol, endCol, found = mc.CorrelatePosition(elem2, lines, nil)
	if !found || lineIdx != 0 || startCol != 11 || endCol != 17 {
		t.Errorf("second link wrong: found=%v line=%d start=%d end=%d", found, lineIdx, startCol, endCol)
	}

	// Third correlation
	lineIdx, startCol, endCol, found = mc.CorrelatePosition(elem3, lines, nil)
	if !found || lineIdx != 1 {
		t.Errorf("third link wrong: found=%v line=%d start=%d end=%d", found, lineIdx, startCol, endCol)
	}
}

func TestMarkerCorrelator_CorrelatesHeadersByLevel(t *testing.T) {
	mc := NewMarkerCorrelator()
	mc.Reset()

	lines := []string{
		HeaderStartMarker(1) + "Title" + HeaderEndMarker,
		HeaderStartMarker(2) + "Section A" + HeaderEndMarker,
		HeaderStartMarker(2) + "Section B" + HeaderEndMarker,
		HeaderStartMarker(1) + "Another Title" + HeaderEndMarker,
	}

	// First H1
	elem := &NavElement{Type: NavElementHeader, Level: 1}
	lineIdx, _, _, found := mc.CorrelatePosition(elem, lines, nil)
	if !found || lineIdx != 0 {
		t.Errorf("first H1 wrong: found=%v line=%d", found, lineIdx)
	}

	// First H2
	elem = &NavElement{Type: NavElementHeader, Level: 2}
	lineIdx, _, _, found = mc.CorrelatePosition(elem, lines, nil)
	if !found || lineIdx != 1 {
		t.Errorf("first H2 wrong: found=%v line=%d", found, lineIdx)
	}

	// Second H2
	elem = &NavElement{Type: NavElementHeader, Level: 2}
	lineIdx, _, _, found = mc.CorrelatePosition(elem, lines, nil)
	if !found || lineIdx != 2 {
		t.Errorf("second H2 wrong: found=%v line=%d", found, lineIdx)
	}

	// Second H1
	elem = &NavElement{Type: NavElementHeader, Level: 1}
	lineIdx, _, _, found = mc.CorrelatePosition(elem, lines, nil)
	if !found || lineIdx != 3 {
		t.Errorf("second H1 wrong: found=%v line=%d", found, lineIdx)
	}
}

func TestMarkerCorrelator_FallsBackWhenNoMarkers(t *testing.T) {
	mc := NewMarkerCorrelator()
	mc.Reset()

	// Lines without any markers - should fall back to ScoringCorrelator
	lines := []string{
		"Some text with Link here",
	}

	elem := &NavElement{Type: NavElementURL, Text: "Link"}
	lineIdx, startCol, endCol, found := mc.CorrelatePosition(elem, lines, nil)

	// Should still find it via fallback
	if !found {
		t.Fatal("expected fallback to find the link")
	}
	if lineIdx != 0 {
		t.Errorf("expected line 0, got %d", lineIdx)
	}
	if startCol < 0 || endCol <= startCol {
		t.Errorf("invalid position: start=%d end=%d", startCol, endCol)
	}
}

func TestDecodeHeaderLevel(t *testing.T) {
	tests := []struct {
		level    int
		expected int
	}{
		{1, 1},
		{2, 2},
		{3, 3},
		{4, 4},
		{5, 5},
		{6, 6},
	}

	for _, tt := range tests {
		marker := HeaderStartMarker(tt.level)
		decoded := DecodeHeaderLevel(marker)
		if decoded != tt.expected {
			t.Errorf("HeaderStartMarker(%d) decoded as %d, expected %d", tt.level, decoded, tt.expected)
		}
	}
}

func TestStripMarkers(t *testing.T) {
	input := "Hello " + LinkStartMarker + "World" + LinkEndMarker + "!"
	expected := "Hello World!"

	result := StripMarkers(input)
	if result != expected {
		t.Errorf("StripMarkers failed: got %q, expected %q", result, expected)
	}
}

func TestIsMarkerRune(t *testing.T) {
	markerRunes := []rune{'\u200B', '\u200C', '\u200D', '\u2060'}
	for _, r := range markerRunes {
		if !IsMarkerRune(r) {
			t.Errorf("IsMarkerRune(%U) should return true", r)
		}
	}

	nonMarkerRunes := []rune{'a', 'Z', '1', ' ', '\n'}
	for _, r := range nonMarkerRunes {
		if IsMarkerRune(r) {
			t.Errorf("IsMarkerRune(%U) should return false", r)
		}
	}
}

func TestMarkerExtractor_WithANSICleaner(t *testing.T) {
	// Simulate ANSI-colored line with markers
	ansiRed := "\x1b[31m"
	ansiReset := "\x1b[0m"
	line := ansiRed + LinkStartMarker + "Red Link" + LinkEndMarker + ansiReset

	cleaner := LineCleanerFunc(func(s string) string {
		// Strip ANSI codes
		return strings.ReplaceAll(strings.ReplaceAll(s, ansiRed, ""), ansiReset, "")
	})

	positions := ExtractAllMarkers([]string{line}, cleaner)

	if len(positions) != 1 {
		t.Fatalf("expected 1 position, got %d", len(positions))
	}

	// Position should be calculated correctly despite ANSI codes
	if positions[0].StartCol != 0 || positions[0].EndCol != 8 {
		t.Errorf("position wrong: start=%d end=%d", positions[0].StartCol, positions[0].EndCol)
	}
}

func TestMarkerCorrelator_IntegrationWithRealRenderer(t *testing.T) {
	// Use the actual ANSI renderer to verify markers work end-to-end
	renderer := NewANSIRenderer()
	result, err := renderer.Render("# Hello\n\nCheck out [Google](https://google.com).")
	if err != nil {
		t.Fatalf("render error: %v", err)
	}

	mc := NewMarkerCorrelator()
	mc.Reset()

	// Extract markers from rendered output
	positions := ExtractAllMarkers(result.Lines, result.Cleaner)

	// Should have at least one header and one link
	hasHeader := false
	hasLink := false
	for _, pos := range positions {
		if pos.Type == MarkerTypeHeader {
			hasHeader = true
		}
		if pos.Type == MarkerTypeLink {
			hasLink = true
		}
	}

	if !hasHeader {
		t.Error("expected to find header marker in rendered output")
	}
	if !hasLink {
		t.Error("expected to find link marker in rendered output")
	}
}

func TestMarkerCorrelator_DuplicateLinkText(t *testing.T) {
	// This is the key test case - same link text appearing multiple times
	mc := NewMarkerCorrelator()
	mc.Reset()

	// Two links with same text "here" but different positions
	lines := []string{
		"Click " + LinkStartMarker + "here" + LinkEndMarker + " or " + LinkStartMarker + "here" + LinkEndMarker + " to continue",
	}

	elem1 := &NavElement{Type: NavElementURL, Text: "here", URL: "https://first.com"}
	elem2 := &NavElement{Type: NavElementURL, Text: "here", URL: "https://second.com"}

	// First "here"
	_, startCol1, _, found1 := mc.CorrelatePosition(elem1, lines, nil)
	if !found1 {
		t.Fatal("first 'here' not found")
	}

	// Second "here" - should get different position
	_, startCol2, _, found2 := mc.CorrelatePosition(elem2, lines, nil)
	if !found2 {
		t.Fatal("second 'here' not found")
	}

	if startCol1 == startCol2 {
		t.Errorf("duplicate link text should have different positions: both at col %d", startCol1)
	}
}
