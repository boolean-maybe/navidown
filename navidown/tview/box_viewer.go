package tview

import (
	"hash/fnv"
	"strconv"
	"strings"

	nav "github.com/boolean-maybe/navidown/navidown"
	"github.com/boolean-maybe/navidown/util"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// boxViewer is a TView adapter for the core navidown markdown session.
// it renders core ANSI output into tview primitives and supports link navigation + activation.
type BoxViewer struct {
	*tview.Box

	core *nav.MarkdownSession

	// displayLines are core rendered lines converted to tview-tagged strings (optional).
	displayLines []string

	// lastContentHash tracks content changes to avoid reprocessing unchanged content.
	lastContentHash uint64

	// ansiConverter is optional. If nil, falls back to tview.TranslateANSI.
	ansiConverter *util.AnsiConverter

	// backgroundColor is the color used to fill empty space. ColorDefault means no fill.
	backgroundColor tcell.Color

	onSelect       func(*BoxViewer, nav.NavElement)
	onStateChanged func(*BoxViewer)

	lastKnownWidth int

	// imageManager handles Kitty image protocol (optional).
	imageManager *ImageManager
}

// newBox creates a new TView markdown viewer backed by a Box.
func NewBox() *BoxViewer {
	box := tview.NewBox()
	box.SetBorder(false)

	return &BoxViewer{
		Box:             box,
		core:            nav.New(nav.Options{}),
		backgroundColor: tcell.ColorDefault,
	}
}

// Core exposes the underlying UI-agnostic markdown session.
func (v *BoxViewer) Core() *nav.MarkdownSession { return v.core }

// SetImageManager enables Kitty image protocol support.
// When set, images in markdown will be rendered as Unicode placeholders.
func (v *BoxViewer) SetImageManager(m *ImageManager) *BoxViewer {
	v.imageManager = m
	v.core.SetImagePostProcessor(NewKittyImageProcessor(m))
	return v
}

// setAnsiConverter configures optional ANSI->tview conversion. If nil, tview.TranslateANSI is used.
func (v *BoxViewer) SetAnsiConverter(c *util.AnsiConverter) {
	v.ansiConverter = c
	v.refreshDisplayCache()
}

// setBackgroundColor sets the background color for empty space.
// use tcell.ColorDefault to disable background filling (default behavior).
func (v *BoxViewer) SetBackgroundColor(color tcell.Color) *BoxViewer {
	v.backgroundColor = color
	return v
}

// setRenderer configures the renderer used by the core viewer.
// this allows dynamic switching between light/dark styles.
func (v *BoxViewer) SetRenderer(r nav.Renderer) *BoxViewer {
	v.core.SetRenderer(r)
	v.refreshDisplayCache()
	return v
}

// setSelectHandler sets the callback for when Enter is pressed on a selected element.
func (v *BoxViewer) SetSelectHandler(handler func(*BoxViewer, nav.NavElement)) *BoxViewer {
	v.onSelect = handler
	return v
}

// setStateChangedHandler sets a callback for when navigation state changes (selection/scroll/history).
func (v *BoxViewer) SetStateChangedHandler(handler func(*BoxViewer)) *BoxViewer {
	v.onStateChanged = handler
	return v
}

// setMarkdown sets markdown content to display.
func (v *BoxViewer) SetMarkdown(content string) *BoxViewer {
	v.ensureWidthConfigured()
	_ = v.core.SetMarkdown(content)
	v.refreshDisplayCache()
	v.fireStateChanged()
	return v
}

func (v *BoxViewer) ensureWidthConfigured() {
	_, _, width, _ := v.GetInnerRect()
	if width > 0 && v.core.CurrentWidth() != width {
		v.lastKnownWidth = width
		v.core.SetWidth(width)
	}
}

// setMarkdownWithSource sets markdown content with source file context.
func (v *BoxViewer) SetMarkdownWithSource(content string, sourceFilePath string, pushToHistory bool) *BoxViewer {
	v.ensureWidthConfigured()
	_ = v.core.SetMarkdownWithSource(content, sourceFilePath, pushToHistory)
	v.refreshDisplayCache()
	v.fireStateChanged()
	return v
}

// draw renders the component.
func (v *BoxViewer) Draw(screen tcell.Screen) {
	v.DrawForSubclass(screen, v)
	x, y, width, height := v.GetInnerRect()
	if width <= 0 || height <= 0 {
		return
	}

	// Check width change
	if width != v.lastKnownWidth {
		v.lastKnownWidth = width
		if v.core.SetWidth(width) {
			v.refreshDisplayCache()
		}
	}

	// fill background if configured
	if v.backgroundColor != tcell.ColorDefault {
		bgStyle := tcell.StyleDefault.Background(v.backgroundColor)
		for row := 0; row < height; row++ {
			for col := 0; col < width; col++ {
				screen.SetContent(x+col, y+row, ' ', nil, bgStyle)
			}
		}
	}

	// selected element info for highlighting.
	selectedLine, highlightStart, highlightEnd := -1, -1, -1
	if sel := v.core.Selected(); sel != nil {
		selectedLine = sel.StartLine
		highlightStart = sel.StartCol
		highlightEnd = sel.EndCol
	}

	// Auto-detect cell pixel dimensions and re-process images if cell size changed
	if v.imageManager != nil {
		if v.imageManager.UpdateCellSize(screen) {
			if v.core.ReprocessImages() {
				v.refreshDisplayCache()
			}
		}
		if v.imageManager.Supported() {
			v.transmitVisibleImages(screen)
		}
	}

	scroll := v.core.ScrollOffset()
	for row := 0; row < height; row++ {
		lineIdx := scroll + row
		if lineIdx < 0 || lineIdx >= len(v.displayLines) {
			break
		}
		line := v.displayLines[lineIdx]
		hs, he := -1, -1
		if lineIdx == selectedLine {
			hs, he = highlightStart, highlightEnd
		}
		v.drawLine(screen, x, y+row, width, line, hs, he, v.backgroundColor)
	}

}

// transmitVisibleImages ensures all known images have been sent to the terminal.
func (v *BoxViewer) transmitVisibleImages(screen tcell.Screen) {
	if v.imageManager == nil {
		return
	}
	v.imageManager.mu.Lock()
	ids := make([]uint32, 0, len(v.imageManager.urlToID))
	for _, id := range v.imageManager.urlToID {
		ids = append(ids, id)
	}
	v.imageManager.mu.Unlock()

	for _, id := range ids {
		v.imageManager.EnsureTransmitted(screen, id)
	}
}

func (v *BoxViewer) refreshDisplayCache() {
	lines := v.core.RenderedLines()
	if len(lines) == 0 {
		v.displayLines = nil
		v.lastContentHash = 0
		return
	}

	// quick hash check - skip expensive conversion if content unchanged
	newHash := hashLines(lines)
	if newHash == v.lastContentHash && v.displayLines != nil {
		return // Content unchanged, skip reprocessing
	}

	v.displayLines = convertDisplayLines(lines, v.ansiConverter)
	v.lastContentHash = newHash
}

// hashLines computes a fast hash of the line slice for cache invalidation.
func hashLines(lines []string) uint64 {
	h := fnv.New64a()
	for _, l := range lines {
		_, _ = h.Write([]byte(l))
	}
	return h.Sum64()
}

func (v *BoxViewer) fireStateChanged() {
	if v.onStateChanged != nil {
		v.onStateChanged(v)
	}
}

func (v *BoxViewer) drawLine(screen tcell.Screen, x, y, width int, line string, highlightStart, highlightEnd int, fillBg tcell.Color) {
	isHighlightLine := highlightStart >= 0 && highlightEnd > highlightStart

	col := 0
	currentFg := tcell.ColorDefault
	currentBg := tcell.ColorDefault
	currentBold := false

	runes := []rune(line)
	for i := 0; i < len(runes) && col < width; {
		// parse tview tag.
		if runes[i] == '[' {
			tagEnd := findTagEnd(runes, i)
			if tagEnd > i {
				tag := string(runes[i+1 : tagEnd])
				fg, bg, bold := parseTag(tag, currentFg, currentBg, currentBold, fillBg)
				currentFg, currentBg, currentBold = fg, bg, bold
				i = tagEnd + 1
				continue
			}
		}

		// use fillBg when currentBg is ColorDefault and fillBg is set
		bg := currentBg
		if bg == tcell.ColorDefault && fillBg != tcell.ColorDefault {
			bg = fillBg
		}

		style := tcell.StyleDefault.Foreground(currentFg).Background(bg)
		if currentBold {
			style = style.Bold(true)
		}
		if isHighlightLine && col >= highlightStart && col < highlightEnd {
			style = style.Reverse(true)
		}

		// Collect combining characters that follow the base rune.
		// This is needed for Kitty image placeholders where U+10EEEE
		// is followed by combining diacritics encoding row/column.
		baseRune := runes[i]
		i++
		var combining []rune
		for i < len(runes) && isCombining(runes[i]) {
			combining = append(combining, runes[i])
			i++
		}

		screen.SetContent(x+col, y, baseRune, combining, style)
		col++
	}
}

// isCombining returns true if the rune is a Unicode combining character.
// This covers the combining diacritical marks used by Kitty image placeholders.
func isCombining(r rune) bool {
	// General combining marks ranges used in Kitty protocol diacritics
	return (r >= 0x0300 && r <= 0x036F) || // Combining Diacritical Marks
		(r >= 0x0483 && r <= 0x0489) || // Cyrillic combining
		(r >= 0x0591 && r <= 0x05C7) || // Hebrew combining
		(r >= 0x0610 && r <= 0x065F) || // Arabic combining
		(r >= 0x06D6 && r <= 0x06ED) || // Arabic combining extended
		(r >= 0x0730 && r <= 0x074A) || // Syriac combining
		(r >= 0x07EB && r <= 0x07F3) || // NKo combining
		(r >= 0x0816 && r <= 0x082D) || // Samaritan combining
		(r >= 0x0951 && r <= 0x0954) || // Devanagari combining
		(r >= 0x0F82 && r <= 0x0F87) || // Tibetan combining
		(r >= 0x135D && r <= 0x135F) || // Ethiopic combining
		(r == 0x17DD) || // Khmer combining
		(r == 0x193A) || // Limbu combining
		(r >= 0x1A17 && r <= 0x1A7F) || // Tai Tham combining
		(r >= 0x1B6B && r <= 0x1B73) || // Balinese combining
		(r >= 0x1CD0 && r <= 0x1CE8) || // Vedic Extensions combining
		(r >= 0x1DC0 && r <= 0x1DFF) || // Combining Diacritical Marks Supplement
		(r >= 0x20D0 && r <= 0x20F0) || // Combining Diacritical Marks for Symbols
		(r >= 0x2CEF && r <= 0x2CF1) || // Coptic combining
		(r >= 0x2DE0 && r <= 0x2DFF) || // Cyrillic Extended-A combining
		(r >= 0xA66F && r <= 0xA69F) || // Cyrillic Extended-B combining
		(r >= 0xA6F0 && r <= 0xA6F1) || // Bamum combining
		(r >= 0xA8E0 && r <= 0xA8F1) || // Devanagari Extended combining
		(r >= 0xAAB0 && r <= 0xAAC1) || // Tai Viet combining
		(r >= 0xFE20 && r <= 0xFE2F) || // Combining Half Marks
		(r == 0x10A0F) || // Kharoshthi combining
		(r == 0x10A38) || // Kharoshthi combining
		(r >= 0x1D185 && r <= 0x1D189) || // Musical Symbols combining
		(r >= 0x1D1AA && r <= 0x1D1AD) || // Musical Symbols combining
		(r >= 0x1D242 && r <= 0x1D244) // Combining Greek Musical Notation
}

func findTagEnd(runes []rune, start int) int {
	for i := start + 1; i < len(runes); i++ {
		if runes[i] == ']' {
			return i
		}
		if runes[i] == '[' {
			return start
		}
	}
	return start
}

func parseTag(tag string, currentFg, currentBg tcell.Color, currentBold bool, fillBg tcell.Color) (tcell.Color, tcell.Color, bool) {
	parts := strings.Split(tag, ":")
	fg, bg, bold := currentFg, currentBg, currentBold

	if len(parts) >= 1 {
		if parts[0] == "-" {
			fg = tcell.ColorDefault
		} else if parts[0] != "" {
			fg = parseColor(parts[0], currentFg)
		}
	}

	if len(parts) >= 2 {
		if parts[1] == "-" {
			// reset to fillBg if set, otherwise ColorDefault
			if fillBg != tcell.ColorDefault {
				bg = fillBg
			} else {
				bg = tcell.ColorDefault
			}
		} else if parts[1] != "" {
			bg = parseColor(parts[1], currentBg)
		}
	}

	if len(parts) >= 3 {
		if strings.Contains(parts[2], "b") {
			bold = true
		} else if parts[2] == "-" {
			bold = false
		}
	}

	return fg, bg, bold
}

func parseColor(s string, fallback tcell.Color) tcell.Color {
	if strings.HasPrefix(s, "#") && len(s) == 7 {
		r, okR := parseHexByte(s[1:3])
		g, okG := parseHexByte(s[3:5])
		b, okB := parseHexByte(s[5:7])
		if okR && okG && okB {
			return tcell.NewRGBColor(r, g, b)
		}
		return fallback
	}
	return tcell.GetColor(s)
}

func parseHexByte(s string) (int32, bool) {
	v, err := strconv.ParseInt(s, 16, 32)
	if err != nil {
		return 0, false
	}
	if v < 0 || v > 255 {
		return 0, false
	}
	return int32(v), true
}
