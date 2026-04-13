package styles

// CatppuccinLatteStyleConfig is the Catppuccin Latte (light) style.
// Ref: https://catppuccin.com/palette
var CatppuccinLatteStyleConfig = BuildStyleConfig(ThemeColors{
	Fg:    "#4c4f69", // text
	Bg:    "#eff1f5", // base
	Muted: "#9ca0b0", // overlay0

	Yellow: "#df8e1d", // yellow
	Orange: "#fe640b", // peach
	Purple: "#8839ef", // mauve
	Green:  "#40a02b", // green
	Blue:   "#1e66f5", // blue
	Cyan:   "#179299", // teal
	Red:    "#d20f39", // red

	ChromaKeyword:      "#8839ef", // mauve for keywords
	ChromaNumber:       "#fe640b", // peach for numbers
	ChromaStringEscape: "#dd7878", // flamingo for escapes
	ChromaPreproc:      "#179299", // teal for preprocessor
	ChromaTag:          "#8839ef", // mauve for tags
	ChromaAttribute:    "#40a02b", // green for attributes
	ChromaDecorator:    "#df8e1d", // yellow for decorators
})
