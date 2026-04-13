package styles

// SolarizedLightStyleConfig is the Solarized Light style.
// Ref: https://ethanschoonover.com/solarized/
var SolarizedLightStyleConfig = BuildStyleConfig(ThemeColors{
	Fg:    "#657b83", // base00
	Bg:    "#fdf6e3", // base3
	Muted: "#93a1a1", // base1

	Yellow: "#b58900", // yellow
	Orange: "#cb4b16", // orange
	Purple: "#6c71c4", // violet
	Green:  "#859900", // green
	Blue:   "#268bd2", // blue
	Cyan:   "#2aa198", // cyan
	Red:    "#dc322f", // red

	ChromaKeyword:      "#859900", // green for keywords (same as Solarized Dark)
	ChromaNumber:       "#d33682", // magenta for numbers
	ChromaStringEscape: "#cb4b16", // orange for escapes
	ChromaPreproc:      "#cb4b16", // orange for preprocessor
	ChromaTag:          "#268bd2", // blue for tags
	ChromaAttribute:    "#b58900", // yellow for attributes
	ChromaDecorator:    "#b58900", // yellow for decorators
})
