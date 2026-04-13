package styles

// MonokaiStyleConfig is the Monokai style.
// Ref: https://monokai.pro/
var MonokaiStyleConfig = BuildStyleConfig(ThemeColors{
	Fg:    "#f8f8f2", // foreground
	Bg:    "#272822", // background
	Muted: "#75715e", // comment

	Yellow: "#e6db74", // yellow (strings)
	Orange: "#fd971f", // orange
	Purple: "#ae81ff", // purple
	Green:  "#a6e22e", // green
	Blue:   "#66d9ef", // cyan (Monokai uses cyan as its blue)
	Cyan:   "#f92672", // red/pink (Monokai's signature accent)
	Red:    "#f92672", // red/pink

	ChromaKeyword:      "#f92672", // red/pink for keywords
	ChromaNumber:       "#ae81ff", // purple for numbers
	ChromaStringEscape: "#ae81ff", // purple for escapes
	ChromaPreproc:      "#f92672", // red/pink for preprocessor
	ChromaTag:          "#f92672", // red/pink for tags
	ChromaAttribute:    "#a6e22e", // green for attributes
	ChromaDecorator:    "#a6e22e", // green for decorators
})
