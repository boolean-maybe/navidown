package styles

// CatppuccinMochaStyleConfig is the Catppuccin Mocha style.
// Ref: https://catppuccin.com/palette
var CatppuccinMochaStyleConfig = BuildStyleConfig(ThemeColors{
	Fg:    "#cdd6f4", // text
	Bg:    "#1e1e2e", // base
	Muted: "#6c7086", // overlay0

	Yellow: "#f9e2af", // yellow
	Orange: "#fab387", // peach
	Purple: "#cba6f7", // mauve
	Green:  "#a6e3a1", // green
	Blue:   "#89b4fa", // blue
	Cyan:   "#94e2d5", // teal
	Red:    "#f38ba8", // red

	ChromaKeyword:      "#cba6f7", // mauve for keywords
	ChromaNumber:       "#fab387", // peach for numbers
	ChromaStringEscape: "#f2cdcd", // flamingo for escapes
	ChromaPreproc:      "#94e2d5", // teal for preprocessor
	ChromaTag:          "#cba6f7", // mauve for tags
	ChromaAttribute:    "#a6e3a1", // green for attributes
	ChromaDecorator:    "#f9e2af", // yellow for decorators
})
