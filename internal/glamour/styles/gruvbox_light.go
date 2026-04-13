package styles

// GruvboxLightStyleConfig is the Gruvbox Light style.
// Ref: https://github.com/morhetz/gruvbox
var GruvboxLightStyleConfig = BuildStyleConfig(ThemeColors{
	Fg:    "#3c3836", // fg (dark0_hard)
	Bg:    "#fbf1c7", // bg0
	Muted: "#928374", // gray

	Yellow: "#b57614", // dark yellow
	Orange: "#af3a03", // dark orange
	Purple: "#8f3f71", // dark purple
	Green:  "#79740e", // dark green
	Blue:   "#076678", // dark blue
	Cyan:   "#427b58", // dark aqua
	Red:    "#9d0006", // dark red

	ChromaKeyword:      "#9d0006", // dark red for keywords
	ChromaNumber:       "#8f3f71", // dark purple for numbers
	ChromaStringEscape: "#af3a03", // dark orange for escapes
	ChromaPreproc:      "#427b58", // dark aqua for preprocessor
	ChromaTag:          "#9d0006", // dark red for tags
	ChromaAttribute:    "#79740e", // dark green for attributes
	ChromaDecorator:    "#79740e", // dark green for decorators
})
