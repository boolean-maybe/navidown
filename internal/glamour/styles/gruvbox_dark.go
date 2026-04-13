package styles

// GruvboxDarkStyleConfig is the Gruvbox Dark style.
// Ref: https://github.com/morhetz/gruvbox
var GruvboxDarkStyleConfig = BuildStyleConfig(ThemeColors{
	Fg:    "#ebdbb2", // fg
	Bg:    "#282828", // bg0
	Muted: "#928374", // gray

	Yellow: "#fabd2f", // bright yellow
	Orange: "#fe8019", // bright orange
	Purple: "#d3869b", // bright purple
	Green:  "#b8bb26", // bright green
	Blue:   "#83a598", // bright blue
	Cyan:   "#8ec07c", // bright aqua
	Red:    "#fb4934", // bright red

	ChromaKeyword:      "#fb4934", // red for keywords
	ChromaNumber:       "#d3869b", // purple for numbers
	ChromaStringEscape: "#fe8019", // orange for escapes
	ChromaPreproc:      "#8ec07c", // aqua for preprocessor
	ChromaTag:          "#fb4934", // red for tags
	ChromaAttribute:    "#b8bb26", // green for attributes
	ChromaDecorator:    "#b8bb26", // green for decorators
})
