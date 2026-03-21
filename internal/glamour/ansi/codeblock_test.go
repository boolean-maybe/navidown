package ansi

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mattn/go-runewidth"
	"github.com/muesli/termenv"
)

func TestBuildBorder(t *testing.T) {
	tests := []struct {
		name        string
		cornerLeft  string
		cornerRight string
		label       string
		width       int
		wantEmpty   bool
		contains    string // substring that must be present (if non-empty)
	}{
		{
			name:        "width too small",
			cornerLeft:  "╭",
			cornerRight: "╮",
			label:       "",
			width:       3,
			wantEmpty:   true,
		},
		{
			name:        "minimum viable width, no label",
			cornerLeft:  "╭",
			cornerRight: "╮",
			label:       "",
			width:       4,
		},
		{
			name:        "no label",
			cornerLeft:  "╭",
			cornerRight: "╮",
			label:       "",
			width:       20,
		},
		{
			name:        "label fits",
			cornerLeft:  "╭",
			cornerRight: "╮",
			label:       "go",
			width:       20,
			contains:    " go ",
		},
		{
			name:        "label exactly fills available space",
			cornerLeft:  "╭",
			cornerRight: "╮",
			label:       "go",
			width:       8,
			contains:    " go ",
		},
		{
			name:        "long label truncated",
			cornerLeft:  "╭",
			cornerRight: "╮",
			label:       "a-very-long-language-name",
			width:       15,
			contains:    "…",
		},
		{
			name:        "label dropped when too narrow",
			cornerLeft:  "╭",
			cornerRight: "╮",
			label:       "go",
			width:       6,
		},
		{
			name:        "bottom border, no label",
			cornerLeft:  "╰",
			cornerRight: "╯",
			label:       "",
			width:       20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildBorder(tt.cornerLeft, tt.cornerRight, tt.label, tt.width)

			if tt.wantEmpty {
				if got != "" {
					t.Fatalf("expected empty string, got %q", got)
				}
				return
			}

			if got == "" {
				t.Fatal("expected non-empty string")
			}

			// core invariant: visual width must equal requested width
			gotWidth := runewidth.StringWidth(got)
			if gotWidth != tt.width {
				t.Errorf("width = %d, want %d; result: %q", gotWidth, tt.width, got)
			}

			if !strings.HasPrefix(got, tt.cornerLeft) {
				t.Errorf("result %q does not start with %q", got, tt.cornerLeft)
			}
			if !strings.HasSuffix(got, tt.cornerRight) {
				t.Errorf("result %q does not end with %q", got, tt.cornerRight)
			}

			if tt.contains != "" {
				if !strings.Contains(got, tt.contains) {
					t.Errorf("result %q does not contain %q", got, tt.contains)
				}
			}
		})
	}
}

func TestWriteCodeLines(t *testing.T) {
	profile := termenv.TrueColor
	noBorder := StylePrimitive{}

	t.Run("pads short lines", func(t *testing.T) {
		var buf bytes.Buffer
		writeCodeLines(&buf, []string{"hi"}, 10, "", "", false, profile, noBorder)
		got := buf.String()
		// "hi" + 8 spaces + newline
		if got != "hi        \n" {
			t.Errorf("unexpected output: %q", got)
		}
	})

	t.Run("truncates long lines", func(t *testing.T) {
		var buf bytes.Buffer
		writeCodeLines(&buf, []string{"abcdefghij"}, 5, "", "", false, profile, noBorder)
		line := strings.TrimRight(buf.String(), "\n")
		visWidth := runewidth.StringWidth(line)
		if visWidth != 5 {
			t.Errorf("expected visual width 5, got %d; line: %q", visWidth, line)
		}
	})

	t.Run("with borders wraps lines", func(t *testing.T) {
		var buf bytes.Buffer
		borderStyle := StylePrimitive{}
		writeCodeLines(&buf, []string{"x"}, 8, "", "", true, profile, borderStyle)
		got := buf.String()
		// should contain border characters
		if !strings.Contains(got, "│") {
			t.Errorf("expected border chars in output: %q", got)
		}
	})

	t.Run("with bgEscape injects escape and reset", func(t *testing.T) {
		var buf bytes.Buffer
		bgEsc := "\x1b[48;2;55;55;55m"
		writeCodeLines(&buf, []string{"code"}, 10, "", bgEsc, false, profile, noBorder)
		got := buf.String()
		if !strings.Contains(got, bgEsc) {
			t.Errorf("expected bg escape in output: %q", got)
		}
		if !strings.Contains(got, ansiFullReset) {
			t.Errorf("expected reset in output: %q", got)
		}
	})

	t.Run("margin prefix prepended", func(t *testing.T) {
		var buf bytes.Buffer
		writeCodeLines(&buf, []string{"x"}, 5, "  ", "", false, profile, noBorder)
		got := buf.String()
		if !strings.HasPrefix(got, "  ") {
			t.Errorf("expected margin prefix, got: %q", got)
		}
	})
}

func TestResolveCodeBlockBg(t *testing.T) {
	strPtr := func(s string) *string { return &s }

	t.Run("ascii profile returns empty", func(t *testing.T) {
		rules := StyleCodeBlock{}
		rules.BackgroundColor = strPtr("#373737")
		got := resolveCodeBlockBg(rules, termenv.Ascii)
		if got != "" {
			t.Errorf("expected empty for ASCII, got %q", got)
		}
	})

	t.Run("prefers chroma background", func(t *testing.T) {
		rules := StyleCodeBlock{}
		rules.BackgroundColor = strPtr("#111111")
		rules.Chroma = &Chroma{
			Background: StylePrimitive{
				BackgroundColor: strPtr("#373737"),
			},
		}
		got := resolveCodeBlockBg(rules, termenv.TrueColor)
		if got != "#373737" {
			t.Errorf("expected #373737, got %q", got)
		}
	})

	t.Run("falls back to rules background", func(t *testing.T) {
		rules := StyleCodeBlock{}
		rules.BackgroundColor = strPtr("#111111")
		got := resolveCodeBlockBg(rules, termenv.TrueColor)
		if got != "#111111" {
			t.Errorf("expected #111111, got %q", got)
		}
	})

	t.Run("returns empty when nothing set", func(t *testing.T) {
		rules := StyleCodeBlock{}
		got := resolveCodeBlockBg(rules, termenv.TrueColor)
		if got != "" {
			t.Errorf("expected empty, got %q", got)
		}
	})
}
