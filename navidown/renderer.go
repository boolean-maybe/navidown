package navidown

import (
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/alecthomas/chroma/v2"
	chromaStyles "github.com/alecthomas/chroma/v2/styles"
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

func stringPtr(v string) *string {
	return &v
}

// NewANSIRenderer creates a renderer with the dark style (backwards compatible default).
func NewANSIRenderer() *ANSIStyleRenderer {
	return NewANSIRendererWithStyle("dark")
}

// NewANSIRendererWithStyle creates a renderer with the specified style.
// styleName can be any key from styles.DefaultStyles (e.g. "dark", "light",
// "dracula", "tokyo-night", "pink") or "auto" to detect from COLORFGBG.
// Unknown names fall back to "dark".
func NewANSIRendererWithStyle(styleName string) *ANSIStyleRenderer {
	var style ansi.StyleConfig

	if styleName == "auto" {
		style = detectStyleFromEnvironment()
	} else if s, ok := styles.DefaultStyles[styleName]; ok {
		style = *s
	} else {
		style = styles.DarkStyleConfig
	}

	// Always clear margins for consistent rendering
	style.Document.Margin = uintPtr(0)
	style.CodeBlock.Margin = uintPtr(0)

	// default code block border color if not specified by the style
	if style.CodeBlock.Color == nil {
		style.CodeBlock.Color = stringPtr("#808080")
	}

	// soften inline code color from bright red to muted steel blue
	style.Code.Color = stringPtr("109")

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

// WithCodeTheme returns a new renderer that uses a named Chroma style for code
// block syntax highlighting, overriding the built-in color map.
// See https://github.com/alecthomas/chroma/tree/master/styles for available names.
func (r *ANSIStyleRenderer) WithCodeTheme(theme string) *ANSIStyleRenderer {
	style := r.glamourStyle
	style.CodeBlock.Theme = theme
	style.CodeBlock.Chroma = nil

	// apply the theme's background color so code blocks get the correct bg
	// even without an explicit WithCodeBackground call
	if cs := chromaStyles.Get(theme); cs != nil {
		if bg := cs.Get(chroma.Background).Background; bg.IsSet() {
			bgStr := bg.String()
			style.CodeBlock.BackgroundColor = &bgStr
		}
	}

	return &ANSIStyleRenderer{
		glamourStyle: style,
		wordWrap:     r.wordWrap,
	}
}

// WithCodeBackground returns a new renderer with the specified code block
// background color (e.g. "#282a36", "236").
func (r *ANSIStyleRenderer) WithCodeBackground(color string) *ANSIStyleRenderer {
	style := r.glamourStyle
	style.CodeBlock.BackgroundColor = &color
	return &ANSIStyleRenderer{
		glamourStyle: style,
		wordWrap:     r.wordWrap,
	}
}

// WithCodeBorder returns a new renderer with the specified code block
// border color (e.g. "#6272a4", "244").
func (r *ANSIStyleRenderer) WithCodeBorder(color string) *ANSIStyleRenderer {
	style := r.glamourStyle
	style.CodeBlock.Color = &color
	return &ANSIStyleRenderer{
		glamourStyle: style,
		wordWrap:     r.wordWrap,
	}
}

// WithWordWrap returns a new renderer with specified word wrap.
func (r *ANSIStyleRenderer) WithWordWrap(cols int) *ANSIStyleRenderer {
	return &ANSIStyleRenderer{
		glamourStyle: r.glamourStyle,
		wordWrap:     cols,
	}
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
