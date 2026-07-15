package navidown

import (
	"os"
	"strings"
	"testing"
)

func TestNormalizeSVGFonts(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "root monospace attribute gets fallback",
			in:   `<svg font-family="Consolas, monospace">`,
			want: `<svg font-family="Consolas, monospace, sans-serif">`,
		},
		{
			name: "bare monospace keyword gets fallback",
			in:   `<text font-family="monospace">x</text>`,
			want: `<text font-family="monospace, sans-serif">x</text>`,
		},
		{
			name: "serif keyword gets fallback",
			in:   `<text font-family="serif">x</text>`,
			want: `<text font-family="serif, sans-serif">x</text>`,
		},
		{
			name: "already has sans-serif is unchanged",
			in:   `<text font-family="Segoe UI, sans-serif">x</text>`,
			want: `<text font-family="Segoe UI, sans-serif">x</text>`,
		},
		{
			name: "explicit DejaVu Sans is unchanged",
			in:   `<text font-family="DejaVu Sans">x</text>`,
			want: `<text font-family="DejaVu Sans">x</text>`,
		},
		{
			name: "inline css font-family gets fallback",
			in:   `<text style="font-family:monospace;fill:#fff">x</text>`,
			want: `<text style="font-family:monospace, sans-serif;fill:#fff">x</text>`,
		},
		{
			name: "css block declaration gets fallback",
			in:   `<style>.a{font-family:Consolas;fill:#000}</style>`,
			want: `<style>.a{font-family:Consolas, sans-serif;fill:#000}</style>`,
		},
		{
			name: "unregistered named family without generic gets fallback",
			in:   `<text font-family="Helvetica Neue">x</text>`,
			want: `<text font-family="Helvetica Neue, sans-serif">x</text>`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(normalizeSVGFonts([]byte(tt.in)))
			if got != tt.want {
				t.Errorf("normalizeSVGFonts()\n got: %s\nwant: %s", got, tt.want)
			}
		})
	}
}

// TestNormalizeSVGFonts_RealDiagrams verifies the two testdata SVGs whose box
// text was blank now carry a sans-serif fallback on the root declaration.
func TestNormalizeSVGFonts_RealDiagrams(t *testing.T) {
	for _, name := range []string{"er-diagram", "class-diagram"} {
		data, err := os.ReadFile("../testdata/svg/" + name + ".svg")
		if err != nil {
			t.Skipf("read %s: %v", name, err)
		}
		out := string(normalizeSVGFonts(data))
		if strings.Contains(out, `font-family="Consolas, monospace"`) {
			t.Errorf("%s: root monospace declaration was not given a sans-serif fallback", name)
		}
		if !strings.Contains(out, "sans-serif") {
			t.Errorf("%s: expected a sans-serif family after normalization", name)
		}
	}
}
