package navidown

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/glamour/styles"
)

// RenderResult is the rendered representation used by the viewer.
type RenderResult struct {
	Lines   []string
	Cleaner LineCleaner
}

// Renderer renders markdown into decorated lines along with a cleaner that can
// remove non-visible decorations from a line for matching and column arithmetic.
type Renderer interface {
	Render(markdown string) (RenderResult, error)
}

// ANSIStyleRenderer renders markdown to ANSI using glamour.
type ANSIStyleRenderer struct {
	glamourStyle ansi.StyleConfig
	wordWrap     int
}

func uintPtr(v uint) *uint {
	return &v
}

// NewANSIRenderer creates a renderer similar to the old pipeline:
// markdown -> glamour ANSI output. By default it disables word-wrapping.
func NewANSIRenderer() *ANSIStyleRenderer {
	style := styles.DarkStyleConfig
	style.Document.Margin = uintPtr(0)
	style.CodeBlock.Margin = uintPtr(0)

	return &ANSIStyleRenderer{
		glamourStyle: style,
		wordWrap:     0,
	}
}

// WithWordWrap configures glamour word wrap (0 means no wrap).
func (r *ANSIStyleRenderer) WithWordWrap(cols int) *ANSIStyleRenderer {
	r.wordWrap = cols
	return r
}

var ansiSGRPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiSGRPattern.ReplaceAllString(s, "")
}

func (r *ANSIStyleRenderer) Render(markdown string) (RenderResult, error) {
	tr, err := glamour.NewTermRenderer(
		glamour.WithStyles(r.glamourStyle),
		glamour.WithWordWrap(r.wordWrap),
	)
	if err != nil {
		return RenderResult{Lines: strings.Split(markdown, "\n"), Cleaner: LineCleanerFunc(func(s string) string { return s })}, err
	}

	out, err := tr.Render(markdown)
	if err != nil {
		return RenderResult{Lines: strings.Split(markdown, "\n"), Cleaner: LineCleanerFunc(func(s string) string { return s })}, err
	}

	lines := strings.Split(out, "\n")
	return RenderResult{
		Lines:   lines,
		Cleaner: LineCleanerFunc(stripANSI),
	}, nil
}
