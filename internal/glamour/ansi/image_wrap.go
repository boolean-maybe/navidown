package ansi

import (
	"strings"

	"github.com/mattn/go-runewidth"
)

// Image placeholder tokens (imageTokenStart .. imageTokenEnd) carry an
// arbitrarily long URL. The ANSI renderer runs several width-sensitive writers
// over the rendered stream — wordwrap.NewWriter (paragraph/heading),
// ansi.Wordwrap (block element, which breaks on '.', ',', '-', etc.) and the
// margin's padding/indent writers — each of which would fracture a long token
// across lines. Once fractured, the downstream image post-processor can no longer
// match the token and it leaks to the screen as raw text (URL included).
//
// To stay intact, every image token is replaced with a run of placeholder runes
// the instant it is emitted (imageTokenTable.mask) and restored once, after the
// entire document has been wrapped and is about to be flushed to the real writer
// (imageTokenTable.restore). Each placeholder rune is one column wide and is not a
// breakpoint, whitespace, or newline, so reflow never breaks within the run; the
// run is sized to the token's display width (clamped below the wrap width) so the
// padding writer reserves exactly the space the restored token will occupy and
// short tokens render byte-for-byte as before — while an over-wide token is
// clamped so it occupies the line without forcing a hard break.
//
// U+E000 is in the BMP Private Use Area, which width libraries consistently
// measure as one column.
const imageTokenPlaceholderRune = '\ue000'
const imageTokenPlaceholder = string(imageTokenPlaceholderRune)

// imageTokenTable records masked image tokens in emission order so they can be
// restored verbatim after wrapping. It hangs off RenderContext via a pointer so
// the by-value context copies passed to Render/Finish share one table.
type imageTokenTable struct {
	tokens []string
}

// mask records token and returns a run of placeholder runes to emit in its place.
// The run width matches the token's display width so downstream padding is
// unchanged, but is clamped to maxWidth-1 so a token wider than the line never
// forces a hard break (maxWidth <= 0 means "no wrap" — use the full width).
func (t *imageTokenTable) mask(token string, maxWidth int) string {
	t.tokens = append(t.tokens, token)

	width := runewidth.StringWidth(token)
	if maxWidth > 0 && width > maxWidth-1 {
		width = maxWidth - 1
	}
	if width < 1 {
		width = 1
	}
	return strings.Repeat(imageTokenPlaceholder, width)
}

// restore replaces each maximal run of placeholder runes in s with the next
// recorded token, in order. It is a no-op when no tokens were masked.
func (t *imageTokenTable) restore(s string) string {
	if t == nil || len(t.tokens) == 0 || !strings.ContainsRune(s, imageTokenPlaceholderRune) {
		return s
	}
	var b strings.Builder
	i := 0
	inRun := false
	for _, r := range s {
		if r == imageTokenPlaceholderRune {
			if !inRun { // first rune of a run consumes one token
				if i < len(t.tokens) {
					b.WriteString(t.tokens[i])
					i++
				}
				inRun = true
			}
			continue // collapse the rest of the run
		}
		inRun = false
		b.WriteRune(r)
	}
	return b.String()
}
