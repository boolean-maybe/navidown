package ansi

import (
	"bytes"
	"strings"
	"testing"

	"github.com/muesli/termenv"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// renderMarkdown renders markdown through the ANSI renderer at the given word-wrap
// width and returns the raw output (including any image placeholder tokens).
func renderMarkdown(t *testing.T, md string, wordWrap int) string {
	t.Helper()
	options := Options{
		WordWrap:     wordWrap,
		ColorProfile: termenv.Ascii, // no color codes — keeps assertions about token bytes clean
	}
	gm := goldmark.New()
	ar := NewRenderer(options)
	gm.SetRenderer(renderer.NewRenderer(renderer.WithNodeRenderers(util.Prioritized(ar, 1000))))

	var buf bytes.Buffer
	if err := gm.Convert([]byte(md), &buf); err != nil {
		t.Fatalf("convert: %v", err)
	}
	return buf.String()
}

// TestImageTokenSurvivesNarrowWordWrap reproduces the bug where a standalone image
// with a long URL and a multi-word alt text is fractured by word-wrap, so the
// downstream post-processor can no longer find an intact ￰IMG:...￱ token
// and the raw token leaks to the screen.
func TestImageTokenSurvivesNarrowWordWrap(t *testing.T) {
	// A realistic mermaid-cache path: ~128 chars, far wider than the 40-col wrap.
	longURL := "/Users/example/Library/Caches/navidown/mermaid/" +
		"8b2bbe8f92b10b5434f7285e7eee6abc1234567890abcdef1234567890abcdef.png"
	md := "![mermaid diagram](" + longURL + ")\n"

	out := renderMarkdown(t, md, 40)

	if !strings.Contains(out, imageTokenStart+"IMG:") {
		t.Fatalf("expected an image token to be emitted, got:\n%q", out)
	}

	// The whole token must appear intact on a single physical line: extract from
	// the start sentinel to the end sentinel and assert no newline splits it.
	start := strings.Index(out, imageTokenStart)
	end := strings.Index(out, imageTokenEnd)
	if start < 0 || end < 0 || end < start {
		t.Fatalf("token sentinels not both present: start=%d end=%d in %q", start, end, out)
	}
	token := out[start : end+len(imageTokenEnd)]
	if strings.ContainsRune(token, '\n') {
		t.Errorf("image token was fractured by word-wrap (contains newline):\n%q", token)
	}

	// And the resolved URL must survive whole inside the token.
	if !strings.Contains(token, longURL) {
		t.Errorf("URL did not survive intact inside token:\ntoken=%q\nwant URL=%q", token, longURL)
	}
}
