package tview

import (
	nav "github.com/boolean-maybe/navidown/navidown"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// InputHandler returns the input handler for this component.
func (v *Viewer) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return v.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
		key := event.Key()

		_, _, _, height := v.GetInnerRect()

		// Alt+Left / Alt+Right history navigation.
		if key == tcell.KeyLeft && event.Modifiers()&tcell.ModAlt != 0 {
			if v.core.GoBack() {
				v.refreshDisplayCache()
				v.fireStateChanged()
				return
			}
		}
		if key == tcell.KeyRight && event.Modifiers()&tcell.ModAlt != 0 {
			if v.core.GoForward() {
				v.refreshDisplayCache()
				v.fireStateChanged()
				return
			}
		}

		switch key {
		case tcell.KeyLeft:
			// Plain Left arrow = go back (fallback for terminals with broken Alt-key support)
			if event.Modifiers() == 0 { // Only if no modifiers
				if v.core.GoBack() {
					v.refreshDisplayCache()
					v.fireStateChanged()
				}
			}
		case tcell.KeyRight:
			// Plain Right arrow = go forward (fallback for terminals with broken Alt-key support)
			if event.Modifiers() == 0 { // Only if no modifiers
				if v.core.GoForward() {
					v.refreshDisplayCache()
					v.fireStateChanged()
				}
			}
		case tcell.KeyUp:
			if v.core.ScrollUp(height) {
				v.fireStateChanged()
			}
		case tcell.KeyDown:
			if v.core.ScrollDown(height) {
				v.fireStateChanged()
			}
		case tcell.KeyPgUp:
			if v.core.PageUp(height) {
				v.fireStateChanged()
			}
		case tcell.KeyPgDn:
			if v.core.PageDown(height) {
				v.fireStateChanged()
			}
		case tcell.KeyHome:
			v.core.Home(height)
			v.fireStateChanged()
		case tcell.KeyEnd:
			v.core.End(height)
			v.fireStateChanged()
		case tcell.KeyTab:
			if v.core.MoveToNextLink(height) {
				v.fireStateChanged()
			}
		case tcell.KeyBacktab:
			if v.core.MoveToPreviousLink(height) {
				v.fireStateChanged()
			}
		case tcell.KeyEnter:
			if v.onSelect != nil {
				if sel := v.core.Selected(); sel != nil {
					// Ensure we pass a stable copy to callback.
					v.onSelect(v, *sel)
					return
				}
			}
		}
	})
}

// SetCorrelator delegates to the core viewer.
func (v *Viewer) SetCorrelator(c nav.PositionCorrelator) {
	v.core.SetCorrelator(c)
}
