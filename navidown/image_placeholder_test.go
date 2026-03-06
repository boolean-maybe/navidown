package navidown

import (
	"testing"
)

func TestFormatAndParseImageToken(t *testing.T) {
	tests := []struct {
		url     string
		alt     string
		wantURL string
		wantAlt string
	}{
		{"https://example.com/img.png", "diagram", "https://example.com/img.png", "diagram"},
		{"./local.png", "", "./local.png", ""},
		{"image.jpg", "photo of cats", "image.jpg", "photo of cats"},
	}

	for _, tt := range tests {
		token := FormatImageToken(tt.url, tt.alt)

		if !ContainsImageToken(token) {
			t.Errorf("ContainsImageToken(%q) = false, want true", token)
		}

		gotURL, gotAlt, ok := ParseImageToken(token)
		if !ok {
			t.Errorf("ParseImageToken failed for url=%q alt=%q", tt.url, tt.alt)
			continue
		}
		if gotURL != tt.wantURL {
			t.Errorf("ParseImageToken url: got %q, want %q", gotURL, tt.wantURL)
		}
		if gotAlt != tt.wantAlt {
			t.Errorf("ParseImageToken alt: got %q, want %q", gotAlt, tt.wantAlt)
		}
	}
}

func TestParseImageToken_Invalid(t *testing.T) {
	tests := []string{
		"not a token",
		"\uFFF0notIMG\uFFF1",
		"",
		"\uFFF0IMG:incomplete",
	}

	for _, s := range tests {
		_, _, ok := ParseImageToken(s)
		if ok {
			t.Errorf("ParseImageToken(%q) should have failed", s)
		}
	}
}

func TestContainsImageToken(t *testing.T) {
	if ContainsImageToken("no image here") {
		t.Error("should not detect image token in plain text")
	}
	if !ContainsImageToken("before" + FormatImageToken("test.png", "alt") + "after") {
		t.Error("should detect image token")
	}
}

func TestImageIDToColor(t *testing.T) {
	tests := []struct {
		id   uint32
		want string
	}{
		{1, "#000001"},
		{255, "#0000ff"},
		{256, "#000100"},
		{0x123456, "#123456"},
		{0, "#000001"}, // 0 is clamped to 1
	}

	for _, tt := range tests {
		got := ImageIDToColor(tt.id)
		if got != tt.want {
			t.Errorf("ImageIDToColor(%d) = %q, want %q", tt.id, got, tt.want)
		}
	}
}

func TestFallbackImageProcessor(t *testing.T) {
	processor := &FallbackImageProcessor{}

	lines := []string{
		"line one",
		FormatImageToken("test.png", "diagram"),
		"line three",
		FormatImageToken("photo.jpg", ""),
	}

	result := processor.ProcessImageTokens(lines, "", 80)

	if len(result) != 4 {
		t.Fatalf("expected 4 lines, got %d", len(result))
	}
	if result[0] != "line one" {
		t.Errorf("line 0: got %q, want %q", result[0], "line one")
	}
	if result[1] != "[image: diagram]" {
		t.Errorf("line 1: got %q, want %q", result[1], "[image: diagram]")
	}
	if result[2] != "line three" {
		t.Errorf("line 2: got %q, want %q", result[2], "line three")
	}
	if result[3] != "[image]" {
		t.Errorf("line 3: got %q, want %q", result[3], "[image]")
	}
}

func TestReplaceImageTokensInLine_MixedContent(t *testing.T) {
	line := "before " + FormatImageToken("test.png", "alt") + " after"
	result := replaceImageTokensInLine(line, func(url, alt string) string {
		return "[" + alt + "]"
	})
	if result != "before [alt] after" {
		t.Errorf("got %q, want %q", result, "before [alt] after")
	}
}

func TestCellDimensions(t *testing.T) {
	tests := []struct {
		name                                       string
		imgW, imgH, maxCols, cellW, cellH, maxRows int
		wantCols, wantRows                         int
	}{
		{"fit within", 80, 40, 80, 8, 16, 0, 10, 2},
		{"scale down", 160, 80, 10, 8, 16, 0, 10, 2},
		{"zero input", 0, 0, 80, 8, 16, 0, 0, 0},
		{"tall image no cap", 80, 320, 80, 8, 16, 0, 10, 20},
		{"tall image capped", 80, 320, 80, 8, 16, 10, 5, 10},
		{"square large capped", 5120, 5120, 80, 8, 16, 20, 40, 20},
		{"wide image capped", 400, 200, 80, 8, 16, 5, 20, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cols, rows := CellDimensions(tt.imgW, tt.imgH, tt.maxCols, tt.cellW, tt.cellH, tt.maxRows)
			if cols != tt.wantCols || rows != tt.wantRows {
				t.Errorf("CellDimensions(%d,%d,%d,%d,%d,%d) = (%d,%d), want (%d,%d)",
					tt.imgW, tt.imgH, tt.maxCols, tt.cellW, tt.cellH, tt.maxRows,
					cols, rows, tt.wantCols, tt.wantRows)
			}
		})
	}
}
