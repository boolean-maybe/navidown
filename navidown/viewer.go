package navidown

import (
	"fmt"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// Viewer is a UI-agnostic navigable markdown state machine.
// It stores rendered lines and link/header positions in terms of the configured renderer.
type Viewer struct {
	// content
	markdown          string
	currentSourceFile string
	renderedLines     []string
	cleaner           LineCleaner
	elements          []NavElement

	// navigation
	selectedIndex int
	scrollOffset  int

	// history
	history *NavigationHistory[PageState]

	// strategies
	renderer   Renderer
	correlator PositionCorrelator
}

// Options configures a Viewer.
type Options struct {
	Renderer   Renderer
	Correlator PositionCorrelator
	HistoryMax int
}

// New creates a new Viewer.
func New(opts Options) *Viewer {
	renderer := opts.Renderer
	if renderer == nil {
		renderer = NewANSIRenderer()
	}
	correlator := opts.Correlator
	if correlator == nil {
		correlator = NewMarkerCorrelator()
	}
	hmax := opts.HistoryMax
	if hmax <= 0 {
		hmax = 50
	}

	return &Viewer{
		selectedIndex: -1,
		scrollOffset:  0,
		history:       NewNavigationHistory[PageState](hmax),
		renderer:      renderer,
		correlator:    correlator,
	}
}

// SetCorrelator sets the correlation strategy.
func (v *Viewer) SetCorrelator(c PositionCorrelator) {
	if c == nil {
		c = NewMarkerCorrelator()
	}
	v.correlator = c
}

// SetRenderer sets the renderer used for new pages.
func (v *Viewer) SetRenderer(r Renderer) {
	if r == nil {
		r = NewANSIRenderer()
	}
	v.renderer = r
}

// Markdown returns the current markdown source.
func (v *Viewer) Markdown() string { return v.markdown }

// SourceFilePath returns the current source file path context.
func (v *Viewer) SourceFilePath() string { return v.currentSourceFile }

// RenderedLines returns all rendered lines.
func (v *Viewer) RenderedLines() []string { return v.renderedLines }

// Elements returns all navigable elements.
func (v *Viewer) Elements() []NavElement { return v.elements }

// VisibleLines returns the rendered lines that should be visible for the given viewport height.
func (v *Viewer) VisibleLines(viewportHeight int) []string {
	if viewportHeight <= 0 || len(v.renderedLines) == 0 {
		return nil
	}
	if v.scrollOffset < 0 {
		v.scrollOffset = 0
	}
	if v.scrollOffset >= len(v.renderedLines) {
		return nil
	}
	end := v.scrollOffset + viewportHeight
	if end > len(v.renderedLines) {
		end = len(v.renderedLines)
	}
	return v.renderedLines[v.scrollOffset:end]
}

// Selected returns a copy of the currently selected element, or nil if none.
// The returned pointer references a copy; modifying it does not affect internal state.
func (v *Viewer) Selected() *NavElement {
	if v.selectedIndex < 0 || v.selectedIndex >= len(v.elements) {
		return nil
	}
	elem := v.elements[v.selectedIndex] // Copy by value
	return &elem
}

// SelectedIndex returns the current selected element index (-1 means none).
func (v *Viewer) SelectedIndex() int { return v.selectedIndex }

// ScrollOffset returns the index of the first visible line.
func (v *Viewer) ScrollOffset() int { return v.scrollOffset }

// SetMarkdown loads markdown. If pushToHistory is true, it stores the current page in back history first.
func (v *Viewer) SetMarkdown(content string) error {
	return v.SetMarkdownWithSource(content, "", false)
}

// SetMarkdownWithSource loads markdown with source file context.
// State is only mutated if rendering succeeds, ensuring the viewer remains valid on error.
func (v *Viewer) SetMarkdownWithSource(content string, sourceFilePath string, pushToHistory bool) error {
	// Parse and render BEFORE mutating state to ensure atomicity
	tmpElements := v.parseMarkdownWithSource([]byte(content), sourceFilePath)
	rendered, err := v.renderer.Render(content)
	if err != nil {
		return err // Nothing mutated, viewer still valid
	}

	// Now safe to push history (current state is still valid)
	if pushToHistory && v.markdown != "" {
		v.history.Push(v.saveCurrentState())
	}

	// Mutate state atomically - all operations succeeded
	v.markdown = content
	v.currentSourceFile = sourceFilePath
	v.elements = tmpElements
	v.renderedLines = rendered.Lines
	v.cleaner = rendered.Cleaner
	if v.cleaner == nil {
		v.cleaner = LineCleanerFunc(func(s string) string { return s })
	}

	v.correlatePositions()

	v.selectedIndex = -1
	v.scrollOffset = 0
	return nil
}

func (v *Viewer) parseMarkdownWithSource(source []byte, sourceFilePath string) []NavElement {
	md := goldmark.New()
	reader := text.NewReader(source)
	doc := md.Parser().Parse(reader)

	var elements []NavElement
	_ = ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch n := node.(type) {
		case *ast.Heading:
			var headingText strings.Builder
			for child := n.FirstChild(); child != nil; child = child.NextSibling() {
				if textNode, ok := child.(*ast.Text); ok {
					headingText.Write(textNode.Segment.Value(source))
				}
			}
			elements = append(elements, NavElement{
				Type:           NavElementHeader,
				Text:           headingText.String(),
				Level:          n.Level,
				SourceFilePath: sourceFilePath,
			})
		case *ast.Link:
			var linkText strings.Builder
			for child := n.FirstChild(); child != nil; child = child.NextSibling() {
				if textNode, ok := child.(*ast.Text); ok {
					linkText.Write(textNode.Segment.Value(source))
				}
			}
			elements = append(elements, NavElement{
				Type:           NavElementURL,
				Text:           linkText.String(),
				URL:            string(n.Destination),
				SourceFilePath: sourceFilePath,
			})
		case *ast.AutoLink:
			elements = append(elements, NavElement{
				Type:           NavElementURL,
				Text:           string(n.URL(source)),
				URL:            string(n.URL(source)),
				SourceFilePath: sourceFilePath,
			})
		}

		return ast.WalkContinue, nil
	})

	return elements
}

func (v *Viewer) correlatePositions() {
	if len(v.elements) == 0 || len(v.renderedLines) == 0 {
		return
	}

	// Reset marker correlator state if applicable
	if mc, ok := v.correlator.(*MarkerCorrelator); ok {
		mc.Reset()
	}

	usedPositions := make(map[string]int)

	type correlation struct {
		found    bool
		elemIdx  int
		lineIdx  int
		startCol int
		endCol   int
		key      string // Computed once to avoid duplicate fmt.Sprintf
	}
	correlations := make([]correlation, len(v.elements))

	for i := range v.elements {
		elem := &v.elements[i]
		lineIdx, startCol, endCol, found := v.correlator.CorrelatePosition(elem, v.renderedLines, v.cleaner)
		if found {
			correlations[i] = correlation{
				found:    true,
				elemIdx:  i,
				lineIdx:  lineIdx,
				startCol: startCol,
				endCol:   endCol,
				key:      fmt.Sprintf("%d:%d", lineIdx, startCol),
			}
		}
	}

	for i := range correlations {
		c := &correlations[i]
		if !c.found {
			continue
		}
		matchLen := c.endCol - c.startCol
		existingLen, exists := usedPositions[c.key]
		if !exists || matchLen > existingLen {
			usedPositions[c.key] = matchLen
		}
	}

	for i := range correlations {
		c := &correlations[i]
		if !c.found {
			continue
		}
		matchLen := c.endCol - c.startCol
		winningLen := usedPositions[c.key]
		if matchLen == winningLen {
			elem := &v.elements[c.elemIdx]
			elem.StartLine = c.lineIdx
			elem.EndLine = c.lineIdx
			elem.StartCol = c.startCol
			elem.EndCol = c.endCol
		}
	}
}

func (v *Viewer) saveCurrentState() PageState {
	elementsCopy := make([]NavElement, len(v.elements))
	copy(elementsCopy, v.elements)

	linesCopy := make([]string, len(v.renderedLines))
	copy(linesCopy, v.renderedLines)

	return PageState{
		Markdown:       v.markdown,
		SourceFilePath: v.currentSourceFile,
		SelectedIndex:  v.selectedIndex,
		ScrollOffset:   v.scrollOffset,
		Elements:       elementsCopy,
		RenderedLines:  linesCopy,
		Cleaner:        v.cleaner,
	}
}

func (v *Viewer) restoreState(state PageState) {
	v.markdown = state.Markdown
	v.currentSourceFile = state.SourceFilePath
	v.scrollOffset = state.ScrollOffset

	v.elements = make([]NavElement, len(state.Elements))
	copy(v.elements, state.Elements)
	v.renderedLines = make([]string, len(state.RenderedLines))
	copy(v.renderedLines, state.RenderedLines)
	v.cleaner = state.Cleaner
	if v.cleaner == nil {
		v.cleaner = LineCleanerFunc(func(s string) string { return s })
	}

	v.selectedIndex = state.SelectedIndex
	if v.selectedIndex < 0 || v.selectedIndex >= len(v.elements) {
		v.selectedIndex = -1
		return
	}
	if v.elements[v.selectedIndex].Type == NavElementHeader {
		v.selectedIndex = -1
	}
}

// CanGoBack returns true if there are pages in the back history.
func (v *Viewer) CanGoBack() bool { return v.history.CanGoBack() }

// CanGoForward returns true if there are pages in the forward history.
func (v *Viewer) CanGoForward() bool { return v.history.CanGoForward() }

// GoBack navigates to the previous page in history.
func (v *Viewer) GoBack() bool {
	if !v.history.CanGoBack() {
		return false
	}

	prevState, ok := v.history.Back()
	if !ok {
		return false
	}

	// Only after we have a valid previous state do we save the current state to forward history.
	v.history.PushToForward(v.saveCurrentState())
	v.restoreState(prevState)
	return true
}

// GoForward navigates to the next page in forward history.
func (v *Viewer) GoForward() bool {
	if !v.history.CanGoForward() {
		return false
	}

	nextState, ok := v.history.Forward()
	if !ok {
		return false
	}

	// Only after we have a valid next state do we save the current state to back history.
	v.history.PushToBack(v.saveCurrentState())
	v.restoreState(nextState)
	return true
}

func (v *Viewer) clearSelectionIfOffScreen(viewportHeight int) {
	if v.selectedIndex < 0 || v.selectedIndex >= len(v.elements) {
		return
	}
	if viewportHeight <= 0 {
		return
	}
	elem := v.elements[v.selectedIndex]

	if elem.EndLine < v.scrollOffset {
		v.selectedIndex = -1
		return
	}
	if elem.StartLine >= v.scrollOffset+viewportHeight {
		v.selectedIndex = -1
		return
	}
}

// ScrollUp scrolls the viewport up by one line.
func (v *Viewer) ScrollUp(viewportHeight int) bool {
	if v.scrollOffset > 0 {
		v.scrollOffset--
		v.clearSelectionIfOffScreen(viewportHeight)
		return true
	}
	return false
}

// ScrollDown scrolls the viewport down by one line.
func (v *Viewer) ScrollDown(viewportHeight int) bool {
	maxOffset := len(v.renderedLines) - viewportHeight
	if maxOffset < 0 {
		maxOffset = 0
	}
	if v.scrollOffset < maxOffset {
		v.scrollOffset++
		v.clearSelectionIfOffScreen(viewportHeight)
		return true
	}
	return false
}

// PageUp scrolls up by one viewport.
func (v *Viewer) PageUp(viewportHeight int) bool {
	moved := false
	for i := 0; i < viewportHeight; i++ {
		if !v.ScrollUp(viewportHeight) {
			break
		}
		moved = true
	}
	return moved
}

// PageDown scrolls down by one viewport.
func (v *Viewer) PageDown(viewportHeight int) bool {
	moved := false
	for i := 0; i < viewportHeight; i++ {
		if !v.ScrollDown(viewportHeight) {
			break
		}
		moved = true
	}
	return moved
}

// Home moves viewport to top.
func (v *Viewer) Home(viewportHeight int) {
	v.scrollOffset = 0
	v.clearSelectionIfOffScreen(viewportHeight)
}

// End moves viewport to bottom.
func (v *Viewer) End(viewportHeight int) {
	maxOffset := len(v.renderedLines) - viewportHeight
	if maxOffset < 0 {
		maxOffset = 0
	}
	v.scrollOffset = maxOffset
	v.clearSelectionIfOffScreen(viewportHeight)
}

func (v *Viewer) ensureVisible(viewportHeight int) {
	if v.selectedIndex < 0 || v.selectedIndex >= len(v.elements) {
		return
	}
	if viewportHeight <= 0 {
		return
	}

	elem := v.elements[v.selectedIndex]
	if elem.StartLine < v.scrollOffset {
		v.scrollOffset = elem.StartLine
	}
	if elem.EndLine >= v.scrollOffset+viewportHeight {
		v.scrollOffset = elem.EndLine - viewportHeight + 1
	}
	if v.scrollOffset < 0 {
		v.scrollOffset = 0
	}
}

// MoveToNextLink moves selection to the next link element.
func (v *Viewer) MoveToNextLink(viewportHeight int) bool {
	if len(v.elements) == 0 {
		return false
	}

	if v.selectedIndex >= 0 {
		for i := v.selectedIndex + 1; i < len(v.elements); i++ {
			if v.elements[i].Type == NavElementURL && v.elements[i].EndCol > v.elements[i].StartCol {
				v.selectedIndex = i
				v.ensureVisible(viewportHeight)
				return true
			}
		}
		return false
	}

	for i := 0; i < len(v.elements); i++ {
		if v.elements[i].Type == NavElementURL &&
			v.elements[i].StartLine >= v.scrollOffset &&
			v.elements[i].StartLine < v.scrollOffset+viewportHeight &&
			v.elements[i].EndCol > v.elements[i].StartCol {
			v.selectedIndex = i
			v.ensureVisible(viewportHeight)
			return true
		}
	}
	return false
}

// MoveToPreviousLink moves selection to the previous link element.
func (v *Viewer) MoveToPreviousLink(viewportHeight int) bool {
	if len(v.elements) == 0 {
		return false
	}

	if v.selectedIndex >= 0 {
		for i := v.selectedIndex - 1; i >= 0; i-- {
			if v.elements[i].Type == NavElementURL && v.elements[i].EndCol > v.elements[i].StartCol {
				v.selectedIndex = i
				v.ensureVisible(viewportHeight)
				return true
			}
		}
		return false
	}

	viewportBottom := v.scrollOffset + viewportHeight - 1
	for i := len(v.elements) - 1; i >= 0; i-- {
		if v.elements[i].Type == NavElementURL &&
			v.elements[i].StartLine >= v.scrollOffset &&
			v.elements[i].StartLine <= viewportBottom &&
			v.elements[i].EndCol > v.elements[i].StartCol {
			v.selectedIndex = i
			v.ensureVisible(viewportHeight)
			return true
		}
	}
	return false
}

// MoveToFirst selects the first link.
func (v *Viewer) MoveToFirst(viewportHeight int) bool {
	if len(v.elements) == 0 {
		return false
	}
	for i := 0; i < len(v.elements); i++ {
		if v.elements[i].Type == NavElementURL && v.elements[i].EndCol > v.elements[i].StartCol {
			if v.selectedIndex != i {
				v.selectedIndex = i
				v.ensureVisible(viewportHeight)
				return true
			}
			return false
		}
	}
	return false
}

// MoveToLast selects the last link.
func (v *Viewer) MoveToLast(viewportHeight int) bool {
	if len(v.elements) == 0 {
		return false
	}
	for i := len(v.elements) - 1; i >= 0; i-- {
		if v.elements[i].Type == NavElementURL && v.elements[i].EndCol > v.elements[i].StartCol {
			if v.selectedIndex != i {
				v.selectedIndex = i
				v.ensureVisible(viewportHeight)
				return true
			}
			return false
		}
	}
	return false
}
