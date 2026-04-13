package styles

import "github.com/boolean-maybe/navidown/internal/glamour/ansi"

// ThemeColors holds the color palette for a theme. BuildStyleConfig uses these
// to produce a full ansi.StyleConfig with consistent structural choices
// (heading=bold, link=underline, blockquote=italic, etc.).
//
// UI-level colors (Fg, Yellow, Green, etc.) typically come from the theme's
// tiki palette. Chroma-specific colors come from the theme's official editor
// syntax highlighting spec.
type ThemeColors struct {
	Fg    string // document text, chroma text, punctuation
	Bg    string // code block background
	Muted string // horizontal rule, chroma comments

	Yellow string // blockquote, emph, literal strings
	Orange string // strong, code block border
	Purple string // headings, name constants, generic subheading
	Green  string // inline code, name function/attribute/decorator, generic inserted
	Blue   string // link, enumeration, keyword type, name/name builtin/name class
	Cyan   string // link text, image, image text
	Red    string // generic deleted, error background

	// chroma-specific overrides
	ChromaKeyword      string // keyword, keyword reserved, keyword namespace, operator
	ChromaNumber       string // literal number (optional — empty = omit)
	ChromaStringEscape string // literal string escape
	ChromaPreproc      string // comment preprocessor
	ChromaTag          string // name tag
	ChromaAttribute    string // name attribute (empty = Green)
	ChromaDecorator    string // name decorator (empty = Green)
}

// BuildStyleConfig constructs an ansi.StyleConfig from a ThemeColors palette.
// Structural decisions (bold headings, underlined links, "│ " blockquotes,
// "# " heading prefixes, etc.) are encoded here once, so individual theme
// files only need to declare colors.
func BuildStyleConfig(c ThemeColors) ansi.StyleConfig {
	attribute := c.Green
	if c.ChromaAttribute != "" {
		attribute = c.ChromaAttribute
	}
	decorator := c.Green
	if c.ChromaDecorator != "" {
		decorator = c.ChromaDecorator
	}

	chroma := &ansi.Chroma{
		Text:           ansi.StylePrimitive{Color: stringPtr(c.Fg)},
		Error:          ansi.StylePrimitive{Color: stringPtr(c.Fg), BackgroundColor: stringPtr(c.Red)},
		Comment:        ansi.StylePrimitive{Color: stringPtr(c.Muted)},
		CommentPreproc: ansi.StylePrimitive{Color: stringPtr(c.ChromaPreproc)},

		Keyword:          ansi.StylePrimitive{Color: stringPtr(c.ChromaKeyword)},
		KeywordReserved:  ansi.StylePrimitive{Color: stringPtr(c.ChromaKeyword)},
		KeywordNamespace: ansi.StylePrimitive{Color: stringPtr(c.ChromaKeyword)},
		KeywordType:      ansi.StylePrimitive{Color: stringPtr(c.Blue)},
		Operator:         ansi.StylePrimitive{Color: stringPtr(c.ChromaKeyword)},

		Punctuation:   ansi.StylePrimitive{Color: stringPtr(c.Fg)},
		Name:          ansi.StylePrimitive{Color: stringPtr(c.Blue)},
		NameBuiltin:   ansi.StylePrimitive{Color: stringPtr(c.Blue)},
		NameTag:       ansi.StylePrimitive{Color: stringPtr(c.ChromaTag)},
		NameAttribute: ansi.StylePrimitive{Color: stringPtr(attribute)},
		NameClass:     ansi.StylePrimitive{Color: stringPtr(c.Blue)},
		NameConstant:  ansi.StylePrimitive{Color: stringPtr(c.Purple)},
		NameDecorator: ansi.StylePrimitive{Color: stringPtr(decorator)},
		NameFunction:  ansi.StylePrimitive{Color: stringPtr(c.Green)},

		LiteralString:       ansi.StylePrimitive{Color: stringPtr(c.Yellow)},
		LiteralStringEscape: ansi.StylePrimitive{Color: stringPtr(c.ChromaStringEscape)},

		GenericDeleted:    ansi.StylePrimitive{Color: stringPtr(c.Red)},
		GenericEmph:       ansi.StylePrimitive{Italic: boolPtr(true)},
		GenericInserted:   ansi.StylePrimitive{Color: stringPtr(c.Green)},
		GenericStrong:     ansi.StylePrimitive{Bold: boolPtr(true)},
		GenericSubheading: ansi.StylePrimitive{Color: stringPtr(c.Purple)},

		Background: ansi.StylePrimitive{BackgroundColor: stringPtr(c.Bg)},
	}

	if c.ChromaNumber != "" {
		chroma.LiteralNumber = ansi.StylePrimitive{Color: stringPtr(c.ChromaNumber)}
	}

	return ansi.StyleConfig{
		Document: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				BlockPrefix: "\n",
				BlockSuffix: "\n",
				Color:       stringPtr(c.Fg),
			},
			Margin: uintPtr(defaultMargin),
		},
		BlockQuote: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color:  stringPtr(c.Yellow),
				Italic: boolPtr(true),
			},
			Indent:      uintPtr(1),
			IndentToken: stringPtr("│ "),
		},
		List: ansi.StyleList{
			StyleBlock: ansi.StyleBlock{
				StylePrimitive: ansi.StylePrimitive{
					Color: stringPtr(c.Fg),
				},
			},
			LevelIndent: defaultListIndent,
		},
		Heading: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				BlockSuffix: "\n",
				Color:       stringPtr(c.Purple),
				Bold:        boolPtr(true),
			},
		},
		H1:            ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Prefix: "# "}},
		H2:            ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Prefix: "## "}},
		H3:            ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Prefix: "### "}},
		H4:            ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Prefix: "#### "}},
		H5:            ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Prefix: "##### "}},
		H6:            ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Prefix: "###### "}},
		Strikethrough: ansi.StylePrimitive{CrossedOut: boolPtr(true)},
		Emph: ansi.StylePrimitive{
			Color:  stringPtr(c.Yellow),
			Italic: boolPtr(true),
		},
		Strong: ansi.StylePrimitive{
			Bold:  boolPtr(true),
			Color: stringPtr(c.Orange),
		},
		HorizontalRule: ansi.StylePrimitive{
			Color:  stringPtr(c.Muted),
			Format: "\n--------\n",
		},
		Item: ansi.StylePrimitive{BlockPrefix: "• "},
		Enumeration: ansi.StylePrimitive{
			BlockPrefix: ". ",
			Color:       stringPtr(c.Blue),
		},
		Task: ansi.StyleTask{
			Ticked:   "[✓] ",
			Unticked: "[ ] ",
		},
		Link: ansi.StylePrimitive{
			Color:     stringPtr(c.Blue),
			Underline: boolPtr(true),
		},
		LinkText: ansi.StylePrimitive{
			Color: stringPtr(c.Cyan),
		},
		Image: ansi.StylePrimitive{
			Color:     stringPtr(c.Blue),
			Underline: boolPtr(true),
		},
		ImageText: ansi.StylePrimitive{
			Color:  stringPtr(c.Cyan),
			Format: "Image: {{.text}} →",
		},
		Code: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: stringPtr(c.Green),
			},
		},
		CodeBlock: ansi.StyleCodeBlock{
			StyleBlock: ansi.StyleBlock{
				StylePrimitive: ansi.StylePrimitive{
					Color: stringPtr(c.Orange),
				},
				Margin: uintPtr(defaultMargin),
			},
			Chroma: chroma,
		},
		Table: ansi.StyleTable{
			StyleBlock: ansi.StyleBlock{
				StylePrimitive: ansi.StylePrimitive{},
			},
		},
		DefinitionDescription: ansi.StylePrimitive{
			BlockPrefix: "\n🠶 ",
		},
	}
}
