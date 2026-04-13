package styles

// OneDarkStyleConfig is the Atom One Dark style.
// Ref: https://github.com/Binaryify/OneDark-Pro
var OneDarkStyleConfig = BuildStyleConfig(ThemeColors{
	Fg:    "#abb2bf", // foreground
	Bg:    "#282c34", // background
	Muted: "#5c6370", // comment

	Yellow: "#e5c07b", // yellow
	Orange: "#d19a66", // orange
	Purple: "#c678dd", // purple
	Green:  "#98c379", // green
	Blue:   "#61afef", // blue
	Cyan:   "#56b6c2", // cyan
	Red:    "#e06c75", // red

	ChromaKeyword:      "#c678dd", // purple for keywords (One Dark convention)
	ChromaNumber:       "#d19a66", // orange for numbers
	ChromaStringEscape: "#56b6c2", // cyan for escapes
	ChromaPreproc:      "#e5c07b", // yellow for preprocessor
	ChromaTag:          "#e06c75", // red for tags
	ChromaAttribute:    "#d19a66", // orange for attributes
	ChromaDecorator:    "#e5c07b", // yellow for decorators
})
