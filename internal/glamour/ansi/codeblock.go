package ansi

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/quick"
	"github.com/alecthomas/chroma/v2/styles"
	xansi "github.com/charmbracelet/x/ansi"
	"github.com/mattn/go-runewidth"
	"github.com/muesli/reflow/ansi"
	"github.com/muesli/reflow/indent"
	"github.com/muesli/termenv"
)

const (
	// The chroma style theme name used for rendering.
	chromaStyleTheme = "charm"

	// The chroma formatter name used for rendering.
	chromaFormatter = "terminal256"

	// ansiFullReset resets all SGR attributes
	ansiFullReset = "\x1b[0m"

	// ansiFgAttrReset resets fg color (39) and text attributes
	// (22=not bold/dim, 23=not italic, 24=not underline, 29=not strikethrough)
	// without resetting background — used to preserve code block background
	ansiFgAttrReset = "\x1b[39;22;23;24;29m"
)

// mutex for synchronizing access to the chroma style registry.
// Related https://github.com/alecthomas/chroma/pull/650
var mutex = sync.Mutex{}

// A CodeBlockElement is used to render code blocks.
type CodeBlockElement struct {
	Code     string
	Language string
}

func chromaStyle(style StylePrimitive) string {
	var s string

	if style.Color != nil {
		s = *style.Color
	}
	if style.BackgroundColor != nil {
		if s != "" {
			s += " "
		}
		s += "bg:" + *style.BackgroundColor
	}
	if style.Italic != nil && *style.Italic {
		if s != "" {
			s += " "
		}
		s += "italic"
	}
	if style.Bold != nil && *style.Bold {
		if s != "" {
			s += " "
		}
		s += "bold"
	}
	if style.Underline != nil && *style.Underline {
		if s != "" {
			s += " "
		}
		s += "underline"
	}

	return s
}

// buildBorder builds a horizontal border line with optional language label.
// cornerLeft/cornerRight are the corner characters (e.g., "╭"/"╮" or "╰"/"╯").
func buildBorder(cornerLeft, cornerRight, label string, width int) string {
	if width < 4 {
		return ""
	}
	inner := width - runewidth.StringWidth(cornerLeft) - runewidth.StringWidth(cornerRight)
	if label != "" {
		// layout: corner + "─" + " label " + "─"*N + corner
		// reserve 1 leading dash + 1 trailing dash = 2 chars for border lines
		labelPart := " " + label + " "
		labelWidth := runewidth.StringWidth(labelPart)
		maxLabelPartWidth := inner - 2
		if maxLabelPartWidth < 3 { // need at least " x " (3 chars)
			labelPart = ""
			labelWidth = 0
		} else if labelWidth > maxLabelPartWidth {
			maxTextWidth := maxLabelPartWidth - 2 // subtract the spaces around label
			label = runewidth.Truncate(label, maxTextWidth, "…")
			labelPart = " " + label + " "
			labelWidth = runewidth.StringWidth(labelPart)
		}
		fill := inner - labelWidth - 1 // -1 for the leading "─"
		return cornerLeft + "─" + labelPart + strings.Repeat("─", fill) + cornerRight
	}
	return cornerLeft + strings.Repeat("─", inner) + cornerRight
}

// writeCodeLines pads each code line to innerWidth and wraps with side borders.
func writeCodeLines(w io.Writer, lines []string, innerWidth int,
	marginPrefix, bgEscape string, hasBorders bool,
	profile termenv.Profile, borderStyle StylePrimitive) {

	var lineBuf bytes.Buffer
	for _, line := range lines {
		lineBuf.Reset()
		visWidth := ansi.PrintableRuneWidth(line)
		if visWidth > innerWidth {
			line = xansi.Truncate(line, innerWidth, "")
			visWidth = innerWidth
		}
		pad := innerWidth - visWidth

		lineBuf.WriteString(marginPrefix)
		if hasBorders {
			renderText(&lineBuf, profile, borderStyle, "│")
		}
		if bgEscape != "" {
			lineBuf.WriteString(bgEscape)
		}
		lineBuf.WriteString(line)
		lineBuf.WriteString(strings.Repeat(" ", pad))
		if bgEscape != "" {
			lineBuf.WriteString(ansiFullReset)
		}
		if hasBorders {
			renderText(&lineBuf, profile, borderStyle, "│")
		}
		_, _ = io.WriteString(w, lineBuf.String())
		_, _ = io.WriteString(w, "\n")
	}
}

// Render renders a CodeBlockElement.
func (e *CodeBlockElement) Render(w io.Writer, ctx RenderContext) error {
	bs := ctx.blockStack
	profile := ctx.options.ColorProfile

	var indentation uint
	var margin uint
	formatter := chromaFormatter
	rules := ctx.options.Styles.CodeBlock
	if rules.Indent != nil {
		indentation = *rules.Indent
	}
	if rules.Margin != nil {
		margin = *rules.Margin
	}
	if len(ctx.options.ChromaFormatter) > 0 {
		formatter = ctx.options.ChromaFormatter
	}
	theme := rules.Theme

	if rules.Chroma != nil && profile != termenv.Ascii {
		theme = chromaStyleTheme
		mutex.Lock()
		_, ok := styles.Registry[theme]
		if !ok {
			styles.Register(chroma.MustNewStyle(theme,
				chroma.StyleEntries{
					chroma.Text:                chromaStyle(rules.Chroma.Text),
					chroma.Error:               chromaStyle(rules.Chroma.Error),
					chroma.Comment:             chromaStyle(rules.Chroma.Comment),
					chroma.CommentPreproc:      chromaStyle(rules.Chroma.CommentPreproc),
					chroma.Keyword:             chromaStyle(rules.Chroma.Keyword),
					chroma.KeywordReserved:     chromaStyle(rules.Chroma.KeywordReserved),
					chroma.KeywordNamespace:    chromaStyle(rules.Chroma.KeywordNamespace),
					chroma.KeywordType:         chromaStyle(rules.Chroma.KeywordType),
					chroma.Operator:            chromaStyle(rules.Chroma.Operator),
					chroma.Punctuation:         chromaStyle(rules.Chroma.Punctuation),
					chroma.Name:                chromaStyle(rules.Chroma.Name),
					chroma.NameBuiltin:         chromaStyle(rules.Chroma.NameBuiltin),
					chroma.NameTag:             chromaStyle(rules.Chroma.NameTag),
					chroma.NameAttribute:       chromaStyle(rules.Chroma.NameAttribute),
					chroma.NameClass:           chromaStyle(rules.Chroma.NameClass),
					chroma.NameConstant:        chromaStyle(rules.Chroma.NameConstant),
					chroma.NameDecorator:       chromaStyle(rules.Chroma.NameDecorator),
					chroma.NameException:       chromaStyle(rules.Chroma.NameException),
					chroma.NameFunction:        chromaStyle(rules.Chroma.NameFunction),
					chroma.NameOther:           chromaStyle(rules.Chroma.NameOther),
					chroma.Literal:             chromaStyle(rules.Chroma.Literal),
					chroma.LiteralNumber:       chromaStyle(rules.Chroma.LiteralNumber),
					chroma.LiteralDate:         chromaStyle(rules.Chroma.LiteralDate),
					chroma.LiteralString:       chromaStyle(rules.Chroma.LiteralString),
					chroma.LiteralStringEscape: chromaStyle(rules.Chroma.LiteralStringEscape),
					chroma.GenericDeleted:      chromaStyle(rules.Chroma.GenericDeleted),
					chroma.GenericEmph:         chromaStyle(rules.Chroma.GenericEmph),
					chroma.GenericInserted:     chromaStyle(rules.Chroma.GenericInserted),
					chroma.GenericStrong:       chromaStyle(rules.Chroma.GenericStrong),
					chroma.GenericSubheading:   chromaStyle(rules.Chroma.GenericSubheading),
					chroma.Background:          chromaStyle(rules.Chroma.Background),
				}))
		}
		mutex.Unlock()
	}

	width := int(bs.Width(ctx)) //nolint:gosec // terminal width is structurally bounded

	// resolve background color for the code block
	bgColor := resolveCodeBlockBg(rules, profile)

	// border style: code block color + background
	borderStyle := StylePrimitive{}
	if rules.Color != nil {
		borderStyle.Color = rules.Color
	}
	if bgColor != "" {
		bgc := bgColor
		borderStyle.BackgroundColor = &bgc
	}

	// render borders for non-ASCII profiles with sufficient width
	hasBorders := profile != termenv.Ascii && width >= 8

	// inner width for code content (subtract 2 for side borders)
	innerWidth := width
	if hasBorders {
		innerWidth = width - 2
	}

	marginPrefix := strings.Repeat(" ", int(indentation+margin)) //nolint:gosec // terminal indent is structurally bounded

	if hasBorders {
		topBorder := buildBorder("╭", "╮", e.Language, width)
		_, _ = io.WriteString(w, marginPrefix)
		renderText(w, profile, borderStyle, topBorder)
		_, _ = io.WriteString(w, "\n")
	}

	// buffer Chroma output so we can process per-line
	var codeBuf bytes.Buffer

	iw := indent.NewWriterPipe(&codeBuf, 0, func(_ io.Writer) {
		renderText(&codeBuf, profile, bs.Current().Style.StylePrimitive, " ")
	})

	// strip trailing newlines from code — it's a markdown parsing artifact
	// and would create an empty bordered line at the end
	code := strings.TrimRight(e.Code, "\r\n")
	// replace tabs with spaces — tabs cause width miscalculation in padding
	code = strings.ReplaceAll(code, "\t", "    ")

	if len(theme) > 0 {
		renderText(iw, profile, bs.Current().Style.StylePrimitive, rules.BlockPrefix)

		err := quick.Highlight(iw, code, e.Language, formatter, theme)
		if err != nil {
			return fmt.Errorf("glamour: error highlighting code: %w", err)
		}
		renderText(iw, profile, bs.Current().Style.StylePrimitive, rules.BlockSuffix)
	} else {
		el := &BaseElement{
			Token: code,
			Style: rules.StylePrimitive,
		}
		if err := el.Render(iw, ctx); err != nil {
			return err
		}
	}

	// build ANSI bg escape for background injection
	bgEscape := ""
	if bgColor != "" {
		bgEscape = fmt.Sprintf("\x1b[%sm", profile.Color(bgColor).Sequence(true))
	}

	// replace full ANSI resets (\x1b[0m) with fg-only + attr-only resets
	// so the background color set at line start persists across Chroma tokens
	codeOutput := codeBuf.String()
	if bgEscape != "" {
		codeOutput = strings.ReplaceAll(codeOutput, ansiFullReset, ansiFgAttrReset)
	}

	codeOutput = strings.ReplaceAll(codeOutput, "\r\n", "\n")
	lines := strings.Split(strings.TrimRight(codeOutput, "\n"), "\n")
	writeCodeLines(w, lines, innerWidth, marginPrefix, bgEscape, hasBorders, profile, borderStyle)

	if hasBorders {
		bottomBorder := buildBorder("╰", "╯", "", width)
		_, _ = io.WriteString(w, marginPrefix)
		renderText(w, profile, borderStyle, bottomBorder)
		_, _ = io.WriteString(w, "\n")
	}

	return nil
}

// resolveCodeBlockBg returns the background color string for the code block.
// Prefers Chroma.Background, falls back to rules.BackgroundColor.
func resolveCodeBlockBg(rules StyleCodeBlock, profile termenv.Profile) string {
	if profile == termenv.Ascii {
		return ""
	}
	if rules.Chroma != nil && rules.Chroma.Background.BackgroundColor != nil {
		return *rules.Chroma.Background.BackgroundColor
	}
	if rules.BackgroundColor != nil {
		return *rules.BackgroundColor
	}
	return ""
}
