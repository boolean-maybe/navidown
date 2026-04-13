package styles

// NordStyleConfig is the Nord style.
// Ref: https://www.nordtheme.com/docs/colors-and-palettes
var NordStyleConfig = BuildStyleConfig(ThemeColors{
	Fg:    "#d8dee9", // nord4 (snow storm)
	Bg:    "#2e3440", // nord0 (polar night)
	Muted: "#4c566a", // nord3

	Yellow: "#ebcb8b", // nord13
	Orange: "#d08770", // nord12
	Purple: "#b48ead", // nord15
	Green:  "#a3be8c", // nord14
	Blue:   "#81a1c1", // nord9
	Cyan:   "#88c0d0", // nord8 (frost)
	Red:    "#bf616a", // nord11

	ChromaKeyword:      "#81a1c1", // nord9 for keywords
	ChromaNumber:       "#b48ead", // nord15 for numbers
	ChromaStringEscape: "#d08770", // nord12 for escapes
	ChromaPreproc:      "#88c0d0", // nord8 for preprocessor
	ChromaTag:          "#81a1c1", // nord9 for tags
	ChromaAttribute:    "#8fbcbb", // nord7 for attributes
	ChromaDecorator:    "#d08770", // nord12 for decorators
})
