package navidown

import (
	"os"
	"strings"
	"testing"
)

func TestParseHexColor(t *testing.T) {
	tests := []struct {
		input   string
		r, g, b uint8
		ok      bool
	}{
		{"#333", 0x33, 0x33, 0x33, true},
		{"#333333", 0x33, 0x33, 0x33, true},
		{"#fff", 0xff, 0xff, 0xff, true},
		{"#FFFFFF", 0xff, 0xff, 0xff, true},
		{"#1565C0", 0x15, 0x65, 0xC0, true},
		{"#000", 0, 0, 0, true},
		{"#000000", 0, 0, 0, true},
		{"", 0, 0, 0, false},
		{"#GG", 0, 0, 0, false},
		{"#12345", 0, 0, 0, false},
		{"red", 0, 0, 0, false},
	}
	for _, tt := range tests {
		r, g, b, ok := parseHexColor(tt.input)
		if ok != tt.ok || r != tt.r || g != tt.g || b != tt.b {
			t.Errorf("parseHexColor(%q) = (%d,%d,%d,%v), want (%d,%d,%d,%v)",
				tt.input, r, g, b, ok, tt.r, tt.g, tt.b, tt.ok)
		}
	}
}

func TestIsDarkGray(t *testing.T) {
	tests := []struct {
		r, g, b uint8
		want    bool
	}{
		{0x33, 0x33, 0x33, true},  // #333 — dark gray
		{0x55, 0x55, 0x55, true},  // #555 — dark gray
		{0x66, 0x66, 0x66, true},  // #666 — dark gray
		{0x44, 0x44, 0x44, true},  // #444 — dark gray
		{0x00, 0x00, 0x00, true},  // black
		{0x88, 0x88, 0x88, false}, // #888 — too light (avg=136 >= 128)
		{0x99, 0x99, 0x99, false}, // #999 — light gray
		{0xcc, 0xcc, 0xcc, false}, // #ccc — light gray
		{0xff, 0xff, 0xff, false}, // white
		{0x15, 0x65, 0xC0, false}, // #1565C0 — blue, has hue
		{0x2E, 0x7D, 0x32, false}, // #2E7D32 — green, has hue
		{0xE6, 0x51, 0x00, false}, // #E65100 — orange, has hue
		{0x7B, 0x1F, 0xA2, false}, // #7B1FA2 — purple, has hue
	}
	for _, tt := range tests {
		got := isDarkGray(tt.r, tt.g, tt.b)
		if got != tt.want {
			t.Errorf("isDarkGray(%02x,%02x,%02x) = %v, want %v", tt.r, tt.g, tt.b, got, tt.want)
		}
	}
}

func TestInvertGray(t *testing.T) {
	tests := []struct {
		r, g, b uint8
		want    string
	}{
		{0x33, 0x33, 0x33, "#cccccc"},
		{0x55, 0x55, 0x55, "#aaaaaa"},
		{0x66, 0x66, 0x66, "#999999"},
		{0x44, 0x44, 0x44, "#bbbbbb"},
		{0x00, 0x00, 0x00, "#ffffff"},
	}
	for _, tt := range tests {
		got := invertGray(tt.r, tt.g, tt.b)
		if got != tt.want {
			t.Errorf("invertGray(%02x,%02x,%02x) = %q, want %q", tt.r, tt.g, tt.b, got, tt.want)
		}
	}
}

func TestRemapDarkColor(t *testing.T) {
	tests := []struct {
		input string
		want  string
		ok    bool
	}{
		{"#333", "#cccccc", true},
		{"#333333", "#cccccc", true},
		{"#555", "#aaaaaa", true},
		{"#666", "#999999", true},
		{"#000", "#ffffff", true},
		{"black", "#e0e0e0", true},
		{"BLACK", "#e0e0e0", true},
		// should NOT remap
		{"#888", "", false},    // avg 136 >= 128, too light
		{"#ccc", "", false},    // light gray
		{"white", "", false},   // not a hex color, not "black"
		{"none", "", false},    // SVG keyword
		{"#1565C0", "", false}, // blue
		{"#2E7D32", "", false}, // green
		{"url(#shadow)", "", false},
	}
	for _, tt := range tests {
		got, ok := remapDarkColor(tt.input)
		if ok != tt.ok || got != tt.want {
			t.Errorf("remapDarkColor(%q) = (%q, %v), want (%q, %v)",
				tt.input, got, ok, tt.want, tt.ok)
		}
	}
}

func TestPreprocessSVGForDarkMode(t *testing.T) {
	input := `<svg>
  <text fill="#333">Title</text>
  <text fill="#1565C0">Colored</text>
  <rect fill="#E3F2FD" stroke="#1565C0"/>
  <polygon fill="#555"/>
  <line stroke="#333" stroke-width="1"/>
  <text fill="black">Dark</text>
  <text fill="#ccc">Light</text>
  <rect fill="url(#grad)"/>
  <text fill="none"/>
</svg>`

	result := string(preprocessSVGForDarkMode([]byte(input)))

	// dark grays should be lightened
	if !strings.Contains(result, `fill="#cccccc"`) {
		t.Error("expected #333 to become #cccccc")
	}
	if !strings.Contains(result, `fill="#aaaaaa"`) {
		t.Error("expected #555 to become #aaaaaa")
	}
	if !strings.Contains(result, `stroke="#cccccc"`) {
		t.Error("expected stroke #333 to become #cccccc")
	}
	if !strings.Contains(result, `fill="#e0e0e0"`) {
		t.Error("expected black to become #e0e0e0")
	}

	// colored fills should NOT change
	if !strings.Contains(result, `fill="#1565C0"`) {
		t.Error("fill #1565C0 should be preserved")
	}
	if !strings.Contains(result, `stroke="#1565C0"`) {
		t.Error("stroke #1565C0 should be preserved")
	}
	if !strings.Contains(result, `fill="#E3F2FD"`) {
		t.Error("fill #E3F2FD should be preserved")
	}

	// light grays, none, and url() should NOT change
	if !strings.Contains(result, `fill="#ccc"`) {
		t.Error("fill #ccc should be preserved")
	}
	if !strings.Contains(result, `fill="url(#grad)"`) {
		t.Error("fill url() should be preserved")
	}
	if !strings.Contains(result, `fill="none"`) {
		t.Error("fill none should be preserved")
	}
}

func TestPreprocessSVGForDarkMode_ArchitectureSVG(t *testing.T) {
	data, err := os.ReadFile("../svg/architecture.svg")
	if err != nil {
		t.Skip("architecture.svg not found:", err)
	}

	result := string(preprocessSVGForDarkMode(data))

	// title text was fill="#333", should now be lightened
	if strings.Contains(result, `fill="#333"`) {
		t.Error("expected all fill=#333 to be remapped")
	}
	// description text was fill="#555", should now be lightened
	if strings.Contains(result, `fill="#555"`) {
		t.Error("expected all fill=#555 to be remapped")
	}

	// colored text inside boxes should be preserved
	if !strings.Contains(result, `fill="#1565C0"`) {
		t.Error("fill #1565C0 (blue text on box) should be preserved")
	}
	if !strings.Contains(result, `fill="#2E7D32"`) {
		t.Error("fill #2E7D32 (green text on box) should be preserved")
	}
	if !strings.Contains(result, `fill="#E65100"`) {
		t.Error("fill #E65100 (orange text on box) should be preserved")
	}
	if !strings.Contains(result, `fill="#7B1FA2"`) {
		t.Error("fill #7B1FA2 (purple text on box) should be preserved")
	}

	// background fills should be preserved
	if !strings.Contains(result, `fill="#E3F2FD"`) {
		t.Error("fill #E3F2FD (box background) should be preserved")
	}
	if !strings.Contains(result, `fill="#FFF3E0"`) {
		t.Error("fill #FFF3E0 (box background) should be preserved")
	}
}
