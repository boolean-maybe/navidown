package navidown

import (
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/boolean-maybe/navidown/internal/glamour"
	"github.com/boolean-maybe/navidown/internal/glamour/ansi"
	"github.com/boolean-maybe/navidown/internal/glamour/styles"
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

// NewANSIRenderer creates a renderer with the dark style (backwards compatible default).
func NewANSIRenderer() *ANSIStyleRenderer {
	return NewANSIRendererWithStyle("dark")
}

// NewANSIRendererWithStyle creates a renderer with the specified style.
// styleName can be "dark", "light", or "auto".
// "auto" uses COLORFGBG environment variable to detect terminal background.
func NewANSIRendererWithStyle(styleName string) *ANSIStyleRenderer {
	var style ansi.StyleConfig

	switch styleName {
	case "light":
		style = styles.LightStyleConfig
	case "auto":
		style = detectStyleFromEnvironment()
	case "dark":
		fallthrough
	default:
		style = styles.DarkStyleConfig
	}

	// Always clear margins for consistent rendering
	style.Document.Margin = uintPtr(0)
	style.CodeBlock.Margin = uintPtr(0)

	return &ANSIStyleRenderer{
		glamourStyle: style,
		wordWrap:     0,
	}
}

// detectStyleFromEnvironment detects terminal theme using COLORFGBG.
// Format: "foreground;background" (e.g., "15;0")
// Background >= 8 indicates light background, < 8 indicates dark.
// Defaults to dark on parse errors or missing variable.
func detectStyleFromEnvironment() ansi.StyleConfig {
	colorfgbg := os.Getenv("COLORFGBG")
	if colorfgbg == "" {
		return styles.DarkStyleConfig
	}

	// Parse format: "foreground;background"
	parts := strings.Split(colorfgbg, ";")
	if len(parts) < 2 {
		return styles.DarkStyleConfig
	}

	// Get last component (background color)
	bgStr := strings.TrimSpace(parts[len(parts)-1])
	bg, err := strconv.Atoi(bgStr)
	if err != nil {
		return styles.DarkStyleConfig
	}

	// Background >= 8 means light background (colors 8-15 are bright)
	if bg >= 8 {
		return styles.LightStyleConfig
	}

	return styles.DarkStyleConfig
}

// WithWordWrap configures glamour word wrap (0 means no wrap).
func (r *ANSIStyleRenderer) WithWordWrap(cols int) *ANSIStyleRenderer {
	r.wordWrap = cols
	return r
}

var ansiSGRPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// stripANSIAndMarkers removes both ANSI escape sequences and marker characters.
func stripANSIAndMarkers(s string) string {
	s = ansiSGRPattern.ReplaceAllString(s, "")
	return StripMarkers(s)
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
		Cleaner: LineCleanerFunc(stripANSIAndMarkers),
	}, nil
}
