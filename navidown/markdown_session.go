package navidown

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// generateSlug converts header text to a URL-safe anchor slug (GitHub-compatible).
// Example: "Hello World" → "hello-world", "What's New?" → "whats-new"
func generateSlug(text string) string {
	var result strings.Builder
	trimmed := strings.TrimSpace(text)

	for _, r := range trimmed {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			result.WriteRune(unicode.ToLower(r))
		case r == ' ':
			result.WriteByte('-')
		case r == '-' || r == '_':
			result.WriteRune(r)
		default:
			// skip punctuation, symbols, emoji, etc.
		}
	}

	return result.String()
}

// MarkdownSession is a UI-agnostic navigable markdown state machine that serves as a model
// for UI components, remembering original Markdown, rendered text, links and scrolling position

// MarkdownSession responsibilities:
// - store markdown, source file path, rendered lines, and nav elements
// - parse markdown to extract navigable elements
// - render markdown via Renderer and keep a cleaner for matching
// - correlate element positions in rendered output
// - track selection, scroll offset, and history
// - expose navigation methods and read accessors
// - accept UI-driven actions to update scroll/selection/history on interaction
type MarkdownSession struct {
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

	// rendering
	currentWidth int // 0 means no wrapping

	// behavior
	alwaysScrollToAnchor bool
}

// Options configures a markdownSession.
type Options struct {
	Renderer   Renderer
	Correlator PositionCorrelator
	HistoryMax int
	// AlwaysScrollToAnchor controls whether ScrollToAnchor scrolls even when
	// the target is already visible. Default false (skip scroll if visible).
	AlwaysScrollToAnchor bool
}

// New creates a new markdownSession.
func New(opts Options) *MarkdownSession {
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

	return &MarkdownSession{
		selectedIndex:        -1,
		scrollOffset:         0,
		history:              NewNavigationHistory[PageState](hmax),
		renderer:             renderer,
		correlator:           correlator,
		alwaysScrollToAnchor: opts.AlwaysScrollToAnchor,
	}
}

// SetCorrelator sets the correlation strategy.
func (v *MarkdownSession) SetCorrelator(c PositionCorrelator) {
	if c == nil {
		c = NewMarkerCorrelator()
	}
	v.correlator = c
}

// SetRenderer sets the renderer used for new pages.
func (v *MarkdownSession) SetRenderer(r Renderer) {
	if r == nil {
		r = NewANSIRenderer()
	}
	v.renderer = r
}

// Markdown returns the current markdown source.
func (v *MarkdownSession) Markdown() string { return v.markdown }

// SourceFilePath returns the current source file path context.
func (v *MarkdownSession) SourceFilePath() string { return v.currentSourceFile }

// CurrentWidth returns the current rendering width (0 means no wrapping).
func (v *MarkdownSession) CurrentWidth() int { return v.currentWidth }

// RenderedLines returns all rendered lines.
func (v *MarkdownSession) RenderedLines() []string { return v.renderedLines }

// Elements returns all navigable elements.
func (v *MarkdownSession) Elements() []NavElement { return v.elements }

// VisibleLines returns the rendered lines that should be visible for the given viewport height.
func (v *MarkdownSession) VisibleLines(viewportHeight int) []string {
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
func (v *MarkdownSession) Selected() *NavElement {
	if v.selectedIndex < 0 || v.selectedIndex >= len(v.elements) {
		return nil
	}
	elem := v.elements[v.selectedIndex] // Copy by value
	return &elem
}

// SelectedIndex returns the current selected element index (-1 means none).
func (v *MarkdownSession) SelectedIndex() int { return v.selectedIndex }

// ScrollOffset returns the index of the first visible line.
func (v *MarkdownSession) ScrollOffset() int { return v.scrollOffset }

// SetWidth updates rendering width and re-renders if changed.
// Returns true if content was re-rendered.
func (v *MarkdownSession) SetWidth(cols int) bool {
	if cols < 0 {
		cols = 0
	}
	if v.currentWidth == cols {
		return false
	}
	if v.markdown == "" {
		v.currentWidth = cols
		return false
	}

	// Save state for scroll restoration after re-render
	selectedElem := v.Selected()
	var anchorElem *NavElement
	if v.scrollOffset >= 0 && v.scrollOffset < len(v.renderedLines) {
		anchorElem = v.findElementNearLine(v.scrollOffset)
		// Don't use anchor element if it had invalid position (from width=0 render).
		// This prevents jumping to wrong scroll position when width changes from 0.
		if anchorElem != nil && anchorElem.EndCol <= anchorElem.StartCol {
			anchorElem = nil
		}
	}

	oldWidth := v.currentWidth
	v.currentWidth = cols

	// Re-render with new width
	if err := v.reRenderWithWidth(cols); err != nil {
		v.currentWidth = oldWidth
		return false
	}

	// Restore state
	v.restoreScrollAndSelection(anchorElem, selectedElem)
	return true
}

func (v *MarkdownSession) reRenderWithWidth(cols int) error {
	rendered, err := v.rendererForWidth(cols).Render(v.markdown)
	if err != nil {
		return err
	}

	v.renderedLines = rendered.Lines
	v.setCleaner(rendered.Cleaner)
	v.correlatePositions()
	return nil
}

func (v *MarkdownSession) rendererForWidth(cols int) Renderer {
	if cols > 0 {
		if ansiRenderer, ok := v.renderer.(*ANSIStyleRenderer); ok {
			return ansiRenderer.WithWordWrap(cols)
		}
	}
	return v.renderer
}

func (v *MarkdownSession) setCleaner(c LineCleaner) {
	v.cleaner = c
	if v.cleaner == nil {
		v.cleaner = LineCleanerFunc(func(s string) string { return s })
	}
}

func (v *MarkdownSession) findElementNearLine(lineIdx int) *NavElement {
	if len(v.elements) == 0 {
		return nil
	}

	// Find element closest to lineIdx.
	// Note: O(n) scan; could use binary search since elements are sorted by StartLine,
	// but element count is typically small enough that this doesn't matter.
	bestIdx := -1
	bestDist := -1
	for i := range v.elements {
		dist := v.elements[i].StartLine - lineIdx
		if dist < 0 {
			dist = -dist
		}
		if bestIdx == -1 || dist < bestDist {
			bestIdx = i
			bestDist = dist
		}
	}

	if bestIdx >= 0 {
		return &v.elements[bestIdx]
	}
	return nil
}

func (v *MarkdownSession) restoreScrollAndSelection(anchorElem, selectedElem *NavElement) {
	if anchorElem != nil {
		for i := range v.elements {
			if v.elementsMatch(&v.elements[i], anchorElem) {
				v.scrollOffset = v.elements[i].StartLine
				break
			}
		}
	}

	if selectedElem != nil && selectedElem.Type == NavElementURL {
		for i := range v.elements {
			if v.elementsMatch(&v.elements[i], selectedElem) {
				v.selectedIndex = i
				return
			}
		}
	}
	// Clear selection if no link was selected, or if the previously selected link
	// could not be found after re-render (e.g., content changed).
	v.selectedIndex = -1

	// Clamp scroll offset to valid range
	if len(v.renderedLines) > 0 && v.scrollOffset >= len(v.renderedLines) {
		v.scrollOffset = len(v.renderedLines) - 1
	}
}

func (v *MarkdownSession) elementsMatch(e1, e2 *NavElement) bool {
	if e1.Type != e2.Type {
		return false
	}
	if e1.Type == NavElementHeader {
		return e1.Slug == e2.Slug
	}
	return e1.URL == e2.URL && e1.Text == e2.Text
}

// SetMarkdown loads markdown. If pushToHistory is true, it stores the current page in back history first.
func (v *MarkdownSession) SetMarkdown(content string) error {
	return v.SetMarkdownWithSource(content, "", false)
}

// SetMarkdownWithSource loads markdown with source file context.
// State is only mutated if rendering succeeds, ensuring the viewer remains valid on error.
func (v *MarkdownSession) SetMarkdownWithSource(content string, sourceFilePath string, pushToHistory bool) error {
	// Parse and render BEFORE mutating state to ensure atomicity
	tmpElements := v.parseMarkdownWithSource([]byte(content), sourceFilePath)

	rendered, err := v.rendererForWidth(v.currentWidth).Render(content)
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
	v.setCleaner(rendered.Cleaner)

	v.correlatePositions()

	v.selectedIndex = -1
	v.scrollOffset = 0
	return nil
}

func (v *MarkdownSession) parseMarkdownWithSource(source []byte, sourceFilePath string) []NavElement {
	md := goldmark.New()
	reader := text.NewReader(source)
	doc := md.Parser().Parse(reader)

	var elements []NavElement
	slugCounts := make(map[string]int)

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
			text := headingText.String()
			baseSlug := generateSlug(text)
			count := slugCounts[baseSlug]
			slugCounts[baseSlug]++

			slug := baseSlug
			if count > 0 {
				slug = fmt.Sprintf("%s-%d", baseSlug, count)
			}

			elements = append(elements, NavElement{
				Type:           NavElementHeader,
				Text:           text,
				Level:          n.Level,
				Slug:           slug,
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

func (v *MarkdownSession) correlatePositions() {
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

func (v *MarkdownSession) saveCurrentState() PageState {
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
		Width:          v.currentWidth,
	}
}

func (v *MarkdownSession) restoreState(state PageState) {
	v.markdown = state.Markdown
	v.currentSourceFile = state.SourceFilePath
	v.scrollOffset = state.ScrollOffset
	v.currentWidth = state.Width

	v.elements = make([]NavElement, len(state.Elements))
	copy(v.elements, state.Elements)
	v.renderedLines = make([]string, len(state.RenderedLines))
	copy(v.renderedLines, state.RenderedLines)
	v.setCleaner(state.Cleaner)

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
func (v *MarkdownSession) CanGoBack() bool { return v.history.CanGoBack() }

// CanGoForward returns true if there are pages in the forward history.
func (v *MarkdownSession) CanGoForward() bool { return v.history.CanGoForward() }

// GoBack navigates to the previous page in history.
func (v *MarkdownSession) GoBack() bool {
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
func (v *MarkdownSession) GoForward() bool {
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

func (v *MarkdownSession) clearSelectionIfOffScreen(viewportHeight int) {
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
func (v *MarkdownSession) ScrollUp(viewportHeight int) bool {
	if v.scrollOffset > 0 {
		v.scrollOffset--
		v.clearSelectionIfOffScreen(viewportHeight)
		return true
	}
	return false
}

// ScrollDown scrolls the viewport down by one line.
func (v *MarkdownSession) ScrollDown(viewportHeight int) bool {
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
func (v *MarkdownSession) PageUp(viewportHeight int) bool {
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
func (v *MarkdownSession) PageDown(viewportHeight int) bool {
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
func (v *MarkdownSession) Home(viewportHeight int) {
	v.scrollOffset = 0
	v.clearSelectionIfOffScreen(viewportHeight)
}

// End moves viewport to bottom.
func (v *MarkdownSession) End(viewportHeight int) {
	maxOffset := len(v.renderedLines) - viewportHeight
	if maxOffset < 0 {
		maxOffset = 0
	}
	v.scrollOffset = maxOffset
	v.clearSelectionIfOffScreen(viewportHeight)
}

func (v *MarkdownSession) ensureVisible(viewportHeight int) {
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
func (v *MarkdownSession) MoveToNextLink(viewportHeight int) bool {
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
func (v *MarkdownSession) MoveToPreviousLink(viewportHeight int) bool {
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
func (v *MarkdownSession) MoveToFirst(viewportHeight int) bool {
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
func (v *MarkdownSession) MoveToLast(viewportHeight int) bool {
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

// FindHeaderBySlug returns the first header element matching the given slug, or nil if not found.
func (v *MarkdownSession) FindHeaderBySlug(slug string) *NavElement {
	for i := range v.elements {
		if v.elements[i].Type == NavElementHeader && v.elements[i].Slug == slug {
			elem := v.elements[i]
			return &elem
		}
	}
	return nil
}

// ScrollToAnchor scrolls to a header by its slug.
// If pushToHistory is true, saves the current position to back history before scrolling.
// Returns true if the header was found (and scrolled to if needed), false otherwise.
func (v *MarkdownSession) ScrollToAnchor(slug string, viewportHeight int, pushToHistory bool) bool {
	header := v.FindHeaderBySlug(slug)
	if header == nil {
		return false
	}

	// Skip scroll if target already visible (unless always-scroll enabled)
	if !v.alwaysScrollToAnchor {
		if header.StartLine >= v.scrollOffset && header.StartLine < v.scrollOffset+viewportHeight {
			return true
		}
	}

	if pushToHistory {
		v.history.Push(v.saveCurrentState())
	}

	v.scrollOffset = header.StartLine
	// Keep selection so Tab can advance to next link after following an anchor

	// clamp to valid range
	maxOffset := len(v.renderedLines) - viewportHeight
	if maxOffset < 0 {
		maxOffset = 0
	}
	if v.scrollOffset > maxOffset {
		v.scrollOffset = maxOffset
	}

	return true
}
