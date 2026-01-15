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

// Viewer is a TView adapter for the core navidown viewer.
// It renders core ANSI output into tview primitives and supports link navigation + activation.
type Viewer struct {
	*tview.Box

	core *nav.Viewer

	// displayLines are core rendered lines converted to tview-tagged strings (optional).
	displayLines []string

	// lastContentHash tracks content changes to avoid reprocessing unchanged content.
	lastContentHash uint64

	// ansiConverter is optional. If nil, falls back to tview.TranslateANSI.
	ansiConverter *util.AnsiConverter

	// backgroundColor is the color used to fill empty space. ColorDefault means no fill.
	backgroundColor tcell.Color

	onSelect       func(*Viewer, nav.NavElement)
	onStateChanged func(*Viewer)
}

// New creates a new TView markdown viewer.
func New() *Viewer {
	box := tview.NewBox()
	box.SetBorder(false)

	return &Viewer{
		Box:             box,
		core:            nav.New(nav.Options{}),
		backgroundColor: tcell.ColorDefault,
	}
}

// Core exposes the underlying UI-agnostic viewer.
func (v *Viewer) Core() *nav.Viewer { return v.core }

// SetAnsiConverter configures optional ANSI->tview conversion. If nil, tview.TranslateANSI is used.
func (v *Viewer) SetAnsiConverter(c *util.AnsiConverter) {
	v.ansiConverter = c
	v.refreshDisplayCache()
}

// SetBackgroundColor sets the background color for empty space.
// Use tcell.ColorDefault to disable background filling (default behavior).
func (v *Viewer) SetBackgroundColor(color tcell.Color) *Viewer {
	v.backgroundColor = color
	return v
}

// SetRenderer configures the renderer used by the core viewer.
// This allows dynamic switching between light/dark styles.
func (v *Viewer) SetRenderer(r nav.Renderer) *Viewer {
	v.core.SetRenderer(r)
	v.refreshDisplayCache()
	return v
}

// SetSelectHandler sets the callback for when Enter is pressed on a selected element.
func (v *Viewer) SetSelectHandler(handler func(*Viewer, nav.NavElement)) *Viewer {
	v.onSelect = handler
	return v
}

// SetStateChangedHandler sets a callback for when navigation state changes (selection/scroll/history).
func (v *Viewer) SetStateChangedHandler(handler func(*Viewer)) *Viewer {
	v.onStateChanged = handler
	return v
}

// SetMarkdown sets markdown content to display.
func (v *Viewer) SetMarkdown(content string) *Viewer {
	_ = v.core.SetMarkdown(content)
	v.refreshDisplayCache()
	v.fireStateChanged()
	return v
}

// SetMarkdownWithSource sets markdown content with source file context.
func (v *Viewer) SetMarkdownWithSource(content string, sourceFilePath string, pushToHistory bool) *Viewer {
	_ = v.core.SetMarkdownWithSource(content, sourceFilePath, pushToHistory)
	v.refreshDisplayCache()
	v.fireStateChanged()
	return v
}

// Draw renders the component.
func (v *Viewer) Draw(screen tcell.Screen) {
	v.Box.DrawForSubclass(screen, v)
	x, y, width, height := v.GetInnerRect()
	if width <= 0 || height <= 0 {
		return
	}

	// Fill background if configured
	if v.backgroundColor != tcell.ColorDefault {
		bgStyle := tcell.StyleDefault.Background(v.backgroundColor)
		for row := 0; row < height; row++ {
			for col := 0; col < width; col++ {
				screen.SetContent(x+col, y+row, ' ', nil, bgStyle)
			}
		}
	}

	// Selected element info for highlighting.
	selectedLine, highlightStart, highlightEnd := -1, -1, -1
	if sel := v.core.Selected(); sel != nil {
		selectedLine = sel.StartLine
		highlightStart = sel.StartCol
		highlightEnd = sel.EndCol
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

func (v *Viewer) refreshDisplayCache() {
	lines := v.core.RenderedLines()
	if len(lines) == 0 {
		v.displayLines = nil
		v.lastContentHash = 0
		return
	}

	// Quick hash check - skip expensive conversion if content unchanged
	newHash := hashLines(lines)
	if newHash == v.lastContentHash && v.displayLines != nil {
		return // Content unchanged, skip reprocessing
	}

	joined := strings.Join(lines, "\n")
	var converted string
	if v.ansiConverter != nil {
		converted = v.ansiConverter.Convert(joined)
	} else {
		converted = tview.TranslateANSI(joined)
	}

	// Strip invisible markers from display lines - they're only used for position calculation
	converted = nav.StripMarkers(converted)

	v.displayLines = strings.Split(converted, "\n")
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

func (v *Viewer) fireStateChanged() {
	if v.onStateChanged != nil {
		v.onStateChanged(v)
	}
}

func (v *Viewer) drawLine(screen tcell.Screen, x, y, width int, line string, highlightStart, highlightEnd int, fillBg tcell.Color) {
	isHighlightLine := highlightStart >= 0 && highlightEnd > highlightStart

	col := 0
	currentFg := tcell.ColorDefault
	currentBg := tcell.ColorDefault
	currentBold := false

	runes := []rune(line)
	for i := 0; i < len(runes) && col < width; {
		// Parse tview tag.
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

		// Use fillBg when currentBg is ColorDefault and fillBg is set
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

		screen.SetContent(x+col, y, runes[i], nil, style)
		col++
		i++
	}
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
			// Reset to fillBg if set, otherwise ColorDefault
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
			return tcell.NewRGBColor(int32(r), int32(g), int32(b))
		}
		return fallback
	}
	return tcell.GetColor(s)
}

func parseHexByte(s string) (int64, bool) {
	v, err := strconv.ParseInt(s, 16, 32)
	if err != nil {
		return 0, false
	}
	return v, true
}
