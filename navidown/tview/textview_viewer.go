package tview

import (
	"strings"

	nav "github.com/boolean-maybe/navidown/navidown"
	"github.com/boolean-maybe/navidown/util"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const selectedRegionID = "navidown_selected"

type selectionKey struct {
	line  int
	start int
	end   int
	ok    bool
}

// textViewViewer is a TextView-based adapter for the core navidown markdown session.
// it preserves TextView paging while supporting link navigation + activation.
type TextViewViewer struct {
	*tview.TextView

	core *nav.MarkdownSession

	// displayLines are core rendered lines converted to tview-tagged strings (optional).
	displayLines []string

	// lastContentHash tracks content changes to avoid reprocessing unchanged content.
	lastContentHash uint64

	// ansiConverter is optional. If nil, falls back to tview.TranslateANSI.
	ansiConverter *util.AnsiConverter

	// backgroundColor is the color used to fill empty space. ColorDefault means no fill.
	backgroundColor tcell.Color

	onSelect       func(*TextViewViewer, nav.NavElement)
	onStateChanged func(*TextViewViewer)

	lastSelection selectionKey
}

// newTextView creates a new TView markdown viewer backed by a TextView.
func NewTextView() *TextViewViewer {
	textView := tview.NewTextView()
	textView.SetBorder(false)
	textView.SetDynamicColors(true)
	textView.SetRegions(true)
	textView.SetWrap(false)
	textView.SetWordWrap(false)

	return &TextViewViewer{
		TextView:        textView,
		core:            nav.New(nav.Options{}),
		backgroundColor: tcell.ColorDefault,
	}
}

// core exposes the underlying UI-agnostic markdown session.
func (v *TextViewViewer) Core() *nav.MarkdownSession { return v.core }

// setAnsiConverter configures optional ANSI->tview conversion. If nil, tview.TranslateANSI is used.
func (v *TextViewViewer) SetAnsiConverter(c *util.AnsiConverter) {
	v.ansiConverter = c
	v.refreshDisplayCache()
}

// setBackgroundColor sets the background color for empty space.
// use tcell.ColorDefault to disable background filling (default behavior).
func (v *TextViewViewer) SetBackgroundColor(color tcell.Color) *TextViewViewer {
	v.backgroundColor = color
	v.TextView.SetBackgroundColor(color)
	return v
}

// setRenderer configures the renderer used by the core viewer.
// this allows dynamic switching between light/dark styles.
func (v *TextViewViewer) SetRenderer(r nav.Renderer) *TextViewViewer {
	v.core.SetRenderer(r)
	v.refreshDisplayCache()
	return v
}

// setSelectHandler sets the callback for when Enter is pressed on a selected element.
func (v *TextViewViewer) SetSelectHandler(handler func(*TextViewViewer, nav.NavElement)) *TextViewViewer {
	v.onSelect = handler
	return v
}

// setStateChangedHandler sets a callback for when navigation state changes (selection/scroll/history).
func (v *TextViewViewer) SetStateChangedHandler(handler func(*TextViewViewer)) *TextViewViewer {
	v.onStateChanged = handler
	return v
}

// setMarkdown sets markdown content to display.
func (v *TextViewViewer) SetMarkdown(content string) *TextViewViewer {
	_ = v.core.SetMarkdown(content)
	v.refreshDisplayCache()
	v.ScrollTo(v.core.ScrollOffset(), 0)
	v.fireStateChanged()
	return v
}

// setMarkdownWithSource sets markdown content with source file context.
func (v *TextViewViewer) SetMarkdownWithSource(content string, sourceFilePath string, pushToHistory bool) *TextViewViewer {
	_ = v.core.SetMarkdownWithSource(content, sourceFilePath, pushToHistory)
	v.refreshDisplayCache()
	v.ScrollTo(v.core.ScrollOffset(), 0)
	v.fireStateChanged()
	return v
}

func (v *TextViewViewer) refreshDisplayCache() {
	lines := v.core.RenderedLines()
	if len(lines) == 0 {
		v.displayLines = nil
		v.lastContentHash = 0
		v.updateTextViewContent(true)
		return
	}

	// quick hash check - skip expensive conversion if content unchanged
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

	// strip invisible markers from display lines - they're only used for position calculation
	converted = nav.StripMarkers(converted)

	v.displayLines = strings.Split(converted, "\n")
	v.lastContentHash = newHash
	v.updateTextViewContent(true)
}

func (v *TextViewViewer) fireStateChanged() {
	if v.onStateChanged != nil {
		v.onStateChanged(v)
	}
}

func (v *TextViewViewer) updateTextViewContent(force bool) {
	current := v.currentSelectionKey()
	if !force && current == v.lastSelection {
		return
	}

	if len(v.displayLines) == 0 {
		v.SetText("")
		v.Highlight()
		v.lastSelection = current
		return
	}

	var builder strings.Builder
	for i, line := range v.displayLines {
		if current.ok && i == current.line && current.end > current.start {
			line = insertRegionTags(line, current.start, current.end, selectedRegionID)
		}
		builder.WriteString(line)
		if i < len(v.displayLines)-1 {
			builder.WriteString("\n")
		}
	}

	v.SetText(builder.String())
	if current.ok && current.end > current.start {
		v.Highlight(selectedRegionID)
	} else {
		v.Highlight()
	}
	v.lastSelection = current
}

func (v *TextViewViewer) currentSelectionKey() selectionKey {
	if sel := v.core.Selected(); sel != nil {
		return selectionKey{
			line:  sel.StartLine,
			start: sel.StartCol,
			end:   sel.EndCol,
			ok:    true,
		}
	}
	return selectionKey{}
}

func insertRegionTags(line string, startCol, endCol int, regionID string) string {
	if startCol < 0 || endCol <= startCol {
		return line
	}

	runes := []rune(line)
	var builder strings.Builder
	col := 0
	insertedStart := false
	insertedEnd := false
	startTag := `["` + regionID + `"]`
	endTag := `[""]`

	for i := 0; i < len(runes); {
		if runes[i] == '[' {
			tagEnd := findTagEnd(runes, i)
			if tagEnd > i {
				builder.WriteString(string(runes[i : tagEnd+1]))
				i = tagEnd + 1
				continue
			}
		}

		if !insertedStart && col == startCol {
			builder.WriteString(startTag)
			insertedStart = true
		}
		if !insertedEnd && col == endCol {
			builder.WriteString(endTag)
			insertedEnd = true
		}

		builder.WriteRune(runes[i])
		i++
		col++
	}

	if !insertedStart {
		return line
	}
	if !insertedEnd {
		builder.WriteString(endTag)
	}
	return builder.String()
}
