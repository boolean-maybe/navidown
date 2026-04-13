package styles

// SolarizedDarkStyleConfig is the Solarized Dark style.
// Ref: https://ethanschoonover.com/solarized/
var SolarizedDarkStyleConfig = BuildStyleConfig(ThemeColors{
	Fg:    "#839496", // base0
	Bg:    "#002b36", // base03
	Muted: "#586e75", // base01

	Yellow: "#b58900", // yellow
	Orange: "#cb4b16", // orange
	Purple: "#6c71c4", // violet
	Green:  "#859900", // green
	Blue:   "#268bd2", // blue
	Cyan:   "#2aa198", // cyan
	Red:    "#dc322f", // red

	ChromaKeyword:      "#859900", // green for keywords (Solarized convention)
	ChromaNumber:       "#d33682", // magenta for numbers
	ChromaStringEscape: "#cb4b16", // orange for escapes
	ChromaPreproc:      "#cb4b16", // orange for preprocessor
	ChromaTag:          "#268bd2", // blue for tags
	ChromaAttribute:    "#b58900", // yellow for attributes
	ChromaDecorator:    "#b58900", // yellow for decorators
})
