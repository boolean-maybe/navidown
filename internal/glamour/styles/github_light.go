package styles

// GithubLightStyleConfig is the GitHub Light style.
// Ref: https://github.com/primer/github-vscode-theme
var GithubLightStyleConfig = BuildStyleConfig(ThemeColors{
	Fg:    "#1f2328", // fg.default
	Bg:    "#f6f8fa", // canvas.subtle (not pure white, to distinguish code)
	Muted: "#656d76", // fg.muted

	Yellow: "#953800", // orange (GitHub Light uses warm tones for emphasis)
	Orange: "#0550ae", // blue accent (GitHub uses blue for strong/emphasis)
	Purple: "#8250df", // purple
	Green:  "#116329", // green
	Blue:   "#0969da", // blue
	Cyan:   "#cf222e", // red (GitHub's link-text accent)
	Red:    "#cf222e", // red

	ChromaKeyword:      "#cf222e", // red for keywords
	ChromaNumber:       "#0550ae", // blue accent for numbers
	ChromaStringEscape: "#0a3069", // dark blue for escapes
	ChromaPreproc:      "#953800", // orange for preprocessor
	ChromaTag:          "#116329", // green for tags
	ChromaAttribute:    "#0550ae", // blue accent for attributes
	ChromaDecorator:    "#8250df", // purple for decorators
})
