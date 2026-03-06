package tview

import (
	"strings"
	"testing"

	nav "github.com/boolean-maybe/navidown/navidown"
)

func TestBuildPlaceholderLines(t *testing.T) {
	placeholder := &nav.ImagePlaceholder{
		ImageID: 1,
		Cols:    3,
		Rows:    2,
		URL:     "test.png",
	}

	lines := BuildPlaceholderLines(placeholder)

	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	for i, line := range lines {
		// Each line should start with a color tag and end with a reset tag
		if !strings.HasPrefix(line, "[#000001]") {
			t.Errorf("line %d: missing color tag prefix, got %q", i, line[:min(20, len(line))])
		}
		if !strings.HasSuffix(line, "[-]") {
			t.Errorf("line %d: missing reset tag suffix", i)
		}

		// Each line should contain placeholder runes
		if !strings.ContainsRune(line, placeholderRune) {
			t.Errorf("line %d: missing placeholder rune U+10EEEE", i)
		}

		// Count placeholder runes (should be 3 per line for 3 cols)
		count := strings.Count(line, string(placeholderRune))
		if count != 3 {
			t.Errorf("line %d: expected 3 placeholder runes, got %d", i, count)
		}
	}
}

func TestBuildPlaceholderLines_Zero(t *testing.T) {
	placeholder := &nav.ImagePlaceholder{
		ImageID: 1,
		Cols:    0,
		Rows:    0,
	}

	lines := BuildPlaceholderLines(placeholder)
	if lines != nil {
		t.Errorf("expected nil for zero dimensions, got %d lines", len(lines))
	}
}

func TestBuildPlaceholderLines_ColorEncoding(t *testing.T) {
	placeholder := &nav.ImagePlaceholder{
		ImageID: 0x00123456,
		Cols:    1,
		Rows:    1,
	}

	lines := BuildPlaceholderLines(placeholder)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}

	// Lower 3 bytes encode in the color tag
	if !strings.HasPrefix(lines[0], "[#123456]") {
		t.Errorf("expected color tag [#123456], got %q", lines[0][:min(15, len(lines[0]))])
	}
}

func TestBuildPlaceholderLines_IDDiacritic(t *testing.T) {
	// ID with non-zero upper byte should include a 3rd diacritic per cell
	placeholder := &nav.ImagePlaceholder{
		ImageID: 0x01010101,
		Cols:    2,
		Rows:    1,
	}

	lines := BuildPlaceholderLines(placeholder)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}

	// Upper byte = 1, so 3rd diacritic should be rowColumnDiacritics[1] = 0x030D
	runes := []rune(lines[0])
	// Skip color tag "[#010101]" = 9 runes
	// First cell: placeholder + row_diac + col_diac + id_diac
	// Find the first placeholder rune
	foundIDDiac := false
	for i, r := range runes {
		if r == 0x030D && i > 0 {
			// Check this isn't a row/col diac by verifying position
			foundIDDiac = true
			break
		}
	}
	if !foundIDDiac {
		t.Error("expected 3rd diacritic U+030D for upper byte 1")
	}

	// ID with zero upper byte should NOT include the 3rd diacritic
	placeholder2 := &nav.ImagePlaceholder{
		ImageID: 0x00010101,
		Cols:    2,
		Rows:    1,
	}
	lines2 := BuildPlaceholderLines(placeholder2)
	line1Runes := len([]rune(lines[0]))
	line2Runes := len([]rune(lines2[0]))
	// With 3rd diacritic: 2 more runes (1 per cell)
	if line1Runes != line2Runes+2 {
		t.Errorf("expected line with ID diacritic to have 2 more runes, got %d vs %d", line1Runes, line2Runes)
	}
}

func TestDiacriticsTable(t *testing.T) {
	// Verify the diacritics table has 297 entries (matching Kitty's gen/rowcolumn-diacritics.txt)
	if len(rowColumnDiacritics) != 297 {
		t.Errorf("expected 297 diacritics, got %d", len(rowColumnDiacritics))
	}

	// Verify first few entries match Kitty spec
	expected := []rune{0x0305, 0x030D, 0x030E, 0x0310, 0x0312}
	for i, want := range expected {
		if rowColumnDiacritics[i] != want {
			t.Errorf("diacritics[%d]: got U+%04X, want U+%04X", i, rowColumnDiacritics[i], want)
		}
	}
}

func TestKittyImageProcessor_NoTokens(t *testing.T) {
	// Use nil manager since no tokens means no resolution needed
	processor := &KittyImageProcessor{}

	lines := []string{"line one", "line two"}
	result := processor.ProcessImageTokens(lines, "", 80)

	if len(result) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(result))
	}
	if result[0] != "line one" || result[1] != "line two" {
		t.Error("lines should pass through unchanged when no tokens present")
	}
}
