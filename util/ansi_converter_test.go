package util

import (
	"fmt"
	"testing"
)

func TestAnsiConverter_Disabled(t *testing.T) {
	c := NewAnsiConverter(false)
	input := "\x1b[1;38;5;196mhello\x1b[0m"
	got := c.Convert(input)
	if got != input {
		t.Errorf("disabled converter should return input unchanged\ngot:  %q\nwant: %q", got, input)
	}
}

func TestAnsiConverter_PlainText(t *testing.T) {
	c := NewAnsiConverter(true)
	input := "hello world"
	got := c.Convert(input)
	if got != input {
		t.Errorf("plain text should pass through unchanged\ngot:  %q\nwant: %q", got, input)
	}
}

func TestAnsiConverter_Reset(t *testing.T) {
	c := NewAnsiConverter(true)

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "explicit reset code 0",
			input: "\x1b[1mbold\x1b[0mreset",
			want:  "[-:-:b]bold[-:-:-]reset",
		},
		{
			name:  "empty params treated as reset",
			input: "\x1b[1mbold\x1b[mreset",
			want:  "[-:-:b]bold[-:-:-]reset",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := c.Convert(tt.input)
			if got != tt.want {
				t.Errorf("got:  %q\nwant: %q", got, tt.want)
			}
		})
	}
}

func TestAnsiConverter_Bold(t *testing.T) {
	c := NewAnsiConverter(true)
	input := "\x1b[1mbold\x1b[22mnormal"
	want := "[-:-:b]bold[-:-:-]normal"
	got := c.Convert(input)
	if got != want {
		t.Errorf("got:  %q\nwant: %q", got, want)
	}
}

func TestAnsiConverter_256Color(t *testing.T) {
	c := NewAnsiConverter(true)

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "foreground 256-color red",
			input: "\x1b[38;5;196mred",
			want:  "[#ff0000:-:-]red",
		},
		{
			name:  "foreground 256-color green",
			input: "\x1b[38;5;46mgreen",
			want:  "[#00ff00:-:-]green",
		},
		{
			name:  "background 256-color blue",
			input: "\x1b[48;5;21mblue",
			want:  "[-:#0000ff:-]blue",
		},
		{
			name:  "grayscale 232",
			input: "\x1b[38;5;232mgray",
			want:  "[#080808:-:-]gray",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := c.Convert(tt.input)
			if got != tt.want {
				t.Errorf("got:  %q\nwant: %q", got, tt.want)
			}
		})
	}
}

func TestAnsiConverter_RGBColor(t *testing.T) {
	c := NewAnsiConverter(true)

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "foreground RGB",
			input: "\x1b[38;2;255;128;0mtext",
			want:  "[#ff8000:-:-]text",
		},
		{
			name:  "background RGB",
			input: "\x1b[48;2;0;255;0mtext",
			want:  "[-:#00ff00:-]text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := c.Convert(tt.input)
			if got != tt.want {
				t.Errorf("got:  %q\nwant: %q", got, tt.want)
			}
		})
	}
}

func TestAnsiConverter_ColorReset(t *testing.T) {
	c := NewAnsiConverter(true)

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "foreground reset code 39",
			input: "\x1b[38;5;196mred\x1b[39mdefault",
			want:  "[#ff0000:-:-]red[-:-:-]default",
		},
		{
			name:  "background reset code 49",
			input: "\x1b[48;5;21mbg\x1b[49mdefault",
			want:  "[-:#0000ff:-]bg[-:-:-]default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := c.Convert(tt.input)
			if got != tt.want {
				t.Errorf("got:  %q\nwant: %q", got, tt.want)
			}
		})
	}
}

func TestAnsiConverter_CombinedSequence(t *testing.T) {
	c := NewAnsiConverter(true)
	// bold + 256-color fg + 256-color bg in one sequence
	input := "\x1b[1;38;5;196;48;5;21mtext\x1b[0m"
	want := "[#ff0000:#0000ff:b]text[-:-:-]"
	got := c.Convert(input)
	if got != want {
		t.Errorf("got:  %q\nwant: %q", got, want)
	}
}

func TestAnsi256ToRGB(t *testing.T) {
	tests := []struct {
		code    int
		r, g, b int
	}{
		{0, 0, 0, 0},         // black
		{1, 128, 0, 0},       // dark red
		{15, 255, 255, 255},  // white
		{16, 0, 0, 0},        // cube start
		{196, 255, 0, 0},     // cube red
		{46, 0, 255, 0},      // cube green
		{21, 0, 0, 255},      // cube blue
		{231, 255, 255, 255}, // cube end
		{232, 8, 8, 8},       // grayscale start
		{255, 238, 238, 238}, // grayscale end
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("code_%d", tt.code), func(t *testing.T) {
			r, g, b := Ansi256ToRGB(tt.code)
			if r != tt.r || g != tt.g || b != tt.b {
				t.Errorf("Ansi256ToRGB(%d) = (%d,%d,%d), want (%d,%d,%d)",
					tt.code, r, g, b, tt.r, tt.g, tt.b)
			}
		})
	}
}

func TestAnsi256ToHex(t *testing.T) {
	tests := []struct {
		code int
		want string
	}{
		{0, "#000000"},
		{196, "#ff0000"},
		{255, "#eeeeee"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("code_%d", tt.code), func(t *testing.T) {
			got := Ansi256ToHex(tt.code)
			if got != tt.want {
				t.Errorf("Ansi256ToHex(%d) = %q, want %q", tt.code, got, tt.want)
			}
		})
	}
}
