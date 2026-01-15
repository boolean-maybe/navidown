// Derived from github.com/charmbracelet/glamour
// Modified by navidown to inject invisible position tracking markers
// Original copyright (c) 2019-2023 Charmbracelet, Inc (see LICENSE)

package ansi

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/muesli/reflow/wordwrap"
)

// Marker constants for header position tracking
// Level is encoded by repeating Word Joiner (U+2060)
const (
	headerMarkerPrefix = "\u200D"       // ZWJ
	headerLevelChar    = "\u2060"       // Word Joiner
	headerEndMarker    = "\u200D\u200C" // ZWJ + ZWNJ
)

// headerStartMarker generates a start marker for a heading with encoded level.
func headerStartMarker(level int) string {
	if level < 1 {
		level = 1
	}
	if level > 6 {
		level = 6
	}
	return headerMarkerPrefix + strings.Repeat(headerLevelChar, level) + headerMarkerPrefix
}

// A HeadingElement is used to render headings.
type HeadingElement struct {
	Level int
	First bool
}

const (
	h1 = iota + 1
	h2
	h3
	h4
	h5
	h6
)

// Render renders a HeadingElement.
func (e *HeadingElement) Render(w io.Writer, ctx RenderContext) error {
	bs := ctx.blockStack
	rules := ctx.options.Styles.Heading

	switch e.Level {
	case h1:
		rules = cascadeStyles(rules, ctx.options.Styles.H1)
	case h2:
		rules = cascadeStyles(rules, ctx.options.Styles.H2)
	case h3:
		rules = cascadeStyles(rules, ctx.options.Styles.H3)
	case h4:
		rules = cascadeStyles(rules, ctx.options.Styles.H4)
	case h5:
		rules = cascadeStyles(rules, ctx.options.Styles.H5)
	case h6:
		rules = cascadeStyles(rules, ctx.options.Styles.H6)
	}

	if !e.First {
		renderText(w, ctx.options.ColorProfile, bs.Current().Style.StylePrimitive, "\n")
	}

	be := BlockElement{
		Block: &bytes.Buffer{},
		Style: cascadeStyle(bs.Current().Style, rules, false),
	}
	bs.Push(be)

	renderText(w, ctx.options.ColorProfile, bs.Parent().Style.StylePrimitive, rules.BlockPrefix)
	renderText(bs.Current().Block, ctx.options.ColorProfile, bs.Current().Style.StylePrimitive, rules.Prefix)

	// Inject start marker with encoded level for position tracking
	if _, err := io.WriteString(bs.Current().Block, headerStartMarker(e.Level)); err != nil {
		return err
	}
	return nil
}

// Finish finishes rendering a HeadingElement.
func (e *HeadingElement) Finish(w io.Writer, ctx RenderContext) error {
	bs := ctx.blockStack

	// Inject end marker for position tracking (before word-wrap processing)
	if _, err := io.WriteString(bs.Current().Block, headerEndMarker); err != nil {
		return err
	}

	rules := bs.Current().Style
	mw := NewMarginWriter(ctx, w, rules)

	flow := wordwrap.NewWriter(int(bs.Width(ctx))) //nolint: gosec
	_, err := flow.Write(bs.Current().Block.Bytes())
	if err != nil {
		return fmt.Errorf("glamour: error writing bytes: %w", err)
	}
	if err := flow.Close(); err != nil {
		return fmt.Errorf("glamour: error closing flow: %w", err)
	}

	_, err = mw.Write(flow.Bytes())
	if err != nil {
		return err
	}

	renderText(w, ctx.options.ColorProfile, bs.Current().Style.StylePrimitive, rules.Suffix)
	renderText(w, ctx.options.ColorProfile, bs.Parent().Style.StylePrimitive, rules.BlockSuffix)

	bs.Current().Block.Reset()
	bs.Pop()
	return nil
}
