package tview

import (
	nav "github.com/boolean-maybe/navidown/navidown"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// inputHandler returns the input handler for this component.
func (v *TextViewViewer) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	tv := v.TextView
	base := tv.InputHandler()
	return v.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
		key := event.Key()

		_, _, _, height := v.GetInnerRect()

		// alt+Left / alt+Right history navigation.
		if key == tcell.KeyLeft && event.Modifiers()&tcell.ModAlt != 0 {
			if v.core.GoBack() {
				v.refreshDisplayCache()
				v.ScrollTo(v.core.ScrollOffset(), 0)
				v.fireStateChanged()
				return
			}
		}
		if key == tcell.KeyRight && event.Modifiers()&tcell.ModAlt != 0 {
			if v.core.GoForward() {
				v.refreshDisplayCache()
				v.ScrollTo(v.core.ScrollOffset(), 0)
				v.fireStateChanged()
				return
			}
		}

		switch key {
		case tcell.KeyLeft:
			// plain Left arrow = go back (fallback for terminals with broken Alt-key support)
			if event.Modifiers() == 0 { // Only if no modifiers
				if v.core.GoBack() {
					v.refreshDisplayCache()
					v.ScrollTo(v.core.ScrollOffset(), 0)
					v.fireStateChanged()
					return
				}
			}
		case tcell.KeyRight:
			// plain Right arrow = go forward (fallback for terminals with broken Alt-key support)
			if event.Modifiers() == 0 { // Only if no modifiers
				if v.core.GoForward() {
					v.refreshDisplayCache()
					v.ScrollTo(v.core.ScrollOffset(), 0)
					v.fireStateChanged()
					return
				}
			}
		case tcell.KeyTab:
			if v.core.MoveToNextLink(height) {
				v.updateTextViewContent(false)
				v.ScrollTo(v.core.ScrollOffset(), 0)
				v.fireStateChanged()
				return
			}
		case tcell.KeyBacktab:
			if v.core.MoveToPreviousLink(height) {
				v.updateTextViewContent(false)
				v.ScrollTo(v.core.ScrollOffset(), 0)
				v.fireStateChanged()
				return
			}
		case tcell.KeyEnter:
			if v.onSelect != nil {
				if sel := v.core.Selected(); sel != nil {
					// ensure we pass a stable copy to callback.
					v.onSelect(v, *sel)
					return
				}
			}
		}

		base(event, setFocus)

		if v.syncCoreScrollFromTextView(height) {
			v.fireStateChanged()
		}
	})
}

func (v *TextViewViewer) syncCoreScrollFromTextView(viewportHeight int) bool {
	desiredRow, _ := v.GetScrollOffset()
	currentRow := v.core.ScrollOffset()
	if desiredRow == currentRow {
		return false
	}

	before := v.currentSelectionKey()
	changed := false

	if desiredRow > currentRow {
		for currentRow < desiredRow {
			if !v.core.ScrollDown(viewportHeight) {
				break
			}
			currentRow++
			changed = true
		}
	} else {
		for currentRow > desiredRow {
			if !v.core.ScrollUp(viewportHeight) {
				break
			}
			currentRow--
			changed = true
		}
	}

	after := v.currentSelectionKey()
	if before != after {
		v.updateTextViewContent(false)
		changed = true
	}

	return changed
}

// setCorrelator delegates to the core viewer.
func (v *TextViewViewer) SetCorrelator(c nav.PositionCorrelator) {
	v.core.SetCorrelator(c)
}
