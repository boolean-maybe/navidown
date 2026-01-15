package navidown

import (
	"errors"
	"testing"
)

func TestViewer_ParsesHeaders(t *testing.T) {
	v := New(Options{Renderer: staticRenderer{lines: []string{"# Header 1", "## Header 2"}}})
	_ = v.SetMarkdown(`# Header 1
Some text
## Header 2`)

	elements := v.Elements()
	if len(elements) != 2 {
		t.Fatalf("Expected 2 headers, got %d", len(elements))
	}

	if elements[0].Type != NavElementHeader || elements[0].Text != "Header 1" || elements[0].Level != 1 {
		t.Fatalf("unexpected first header: %#v", elements[0])
	}
	if elements[1].Type != NavElementHeader || elements[1].Text != "Header 2" || elements[1].Level != 2 {
		t.Fatalf("unexpected second header: %#v", elements[1])
	}
}

func TestViewer_ParsesLinks(t *testing.T) {
	v := New(Options{Renderer: staticRenderer{lines: []string{"Title", "Google", "GitHub"}}})
	_ = v.SetMarkdown(`# Title
Check out [Google](https://google.com) and [GitHub](https://github.com).`)

	elements := v.Elements()
	if len(elements) != 3 {
		t.Fatalf("Expected 3 elements, got %d", len(elements))
	}
	if elements[1].Type != NavElementURL || elements[1].Text != "Google" || elements[1].URL != "https://google.com" {
		t.Fatalf("unexpected first link: %#v", elements[1])
	}
	if elements[2].Type != NavElementURL || elements[2].Text != "GitHub" || elements[2].URL != "https://github.com" {
		t.Fatalf("unexpected second link: %#v", elements[2])
	}
}

func TestViewer_TabTraversesLinksOnly(t *testing.T) {
	v := New(Options{Renderer: staticRenderer{lines: []string{
		"# Header 1",
		"Link 1",
		"## Header 2",
		"Link 2",
	}}})
	_ = v.SetMarkdown(`# Header 1
[Link 1](https://example1.com)
## Header 2
[Link 2](https://example2.com)`)

	if v.SelectedIndex() != -1 {
		t.Fatalf("initial selection should be -1, got %d", v.SelectedIndex())
	}

	if !v.MoveToNextLink(10) {
		t.Fatal("expected to select first link")
	}
	if v.Selected() == nil || v.Selected().Text != "Link 1" {
		t.Fatalf("expected Link 1 selected, got %#v", v.Selected())
	}

	if !v.MoveToNextLink(10) {
		t.Fatal("expected to move to second link")
	}
	if v.Selected() == nil || v.Selected().Text != "Link 2" {
		t.Fatalf("expected Link 2 selected, got %#v", v.Selected())
	}
}

func TestViewer_HistoryPreservesSelection(t *testing.T) {
	r1 := staticRenderer{lines: []string{"Link A", "Link B", "Link C"}}
	v := New(Options{Renderer: r1})

	_ = v.SetMarkdownWithSource("# Page 1\n[Link A](a.md)\n[Link B](b.md)\n[Link C](c.md)", "page1.md", false)
	v.MoveToNextLink(10) // A
	v.MoveToNextLink(10) // B
	v.MoveToNextLink(10) // C
	if v.Selected() == nil || v.Selected().Text != "Link C" {
		t.Fatalf("expected Link C selected, got %#v", v.Selected())
	}

	// Swap renderer for page 2 so correlation still yields non-zero spans.
	v.SetRenderer(staticRenderer{lines: []string{"Link X"}})
	_ = v.SetMarkdownWithSource("# Page 2\n[Link X](x.md)", "page2.md", true)
	if !v.GoBack() {
		t.Fatal("expected GoBack to succeed")
	}

	if v.Selected() == nil || v.Selected().Text != "Link C" {
		t.Fatalf("expected Link C restored, got %#v", v.Selected())
	}
}

func TestViewer_SkipsZeroWidthLinksInBackwardAndJumpNavigation(t *testing.T) {
	v := New(Options{Renderer: staticRenderer{lines: []string{"x"}}})

	// Simulate two links where the second is degenerate (not correlated / zero width).
	v.elements = []NavElement{
		{Type: NavElementURL, Text: "good", URL: "good.md", StartLine: 0, EndLine: 0, StartCol: 0, EndCol: 4},
		{Type: NavElementURL, Text: "bad", URL: "bad.md", StartLine: 0, EndLine: 0, StartCol: 0, EndCol: 0},
	}

	// Jump operations should skip the zero-width one.
	if !v.MoveToFirst(10) {
		t.Fatal("expected MoveToFirst to succeed")
	}
	if v.Selected() == nil || v.Selected().Text != "good" {
		t.Fatalf("expected 'good' selected, got %#v", v.Selected())
	}

	// MoveToLast should also pick the last valid link, not the degenerate one.
	v.selectedIndex = -1
	if !v.MoveToLast(10) {
		t.Fatal("expected MoveToLast to succeed")
	}
	if v.Selected() == nil || v.Selected().Text != "good" {
		t.Fatalf("expected 'good' selected, got %#v", v.Selected())
	}

	// Backward navigation should skip the degenerate one.
	v.selectedIndex = 1 // pretend we were on the degenerate one
	if !v.MoveToPreviousLink(10) {
		t.Fatal("expected MoveToPreviousLink to succeed")
	}
	if v.Selected() == nil || v.Selected().Text != "good" {
		t.Fatalf("expected 'good' selected, got %#v", v.Selected())
	}
}

type fixedCorrelator struct {
	lineIdx  int
	startCol int
	endCol   int
	found    bool
}

func (c fixedCorrelator) CorrelatePosition(_ *NavElement, _ []string, _ LineCleaner) (int, int, int, bool) {
	return c.lineIdx, c.startCol, c.endCol, c.found
}

func TestViewer_CorrelatePositions_DoesNotDropLine0Col0Matches(t *testing.T) {
	v := New(Options{Renderer: staticRenderer{lines: []string{"x"}}})
	v.correlator = fixedCorrelator{lineIdx: 0, startCol: 0, endCol: 0, found: true}
	v.renderedLines = []string{"anything"}
	v.cleaner = LineCleanerFunc(func(s string) string { return s })

	v.elements = []NavElement{
		{Type: NavElementURL, Text: "x", StartLine: -1, EndLine: -1, StartCol: -1, EndCol: -1},
	}

	v.correlatePositions()

	if v.elements[0].StartLine != 0 || v.elements[0].StartCol != 0 || v.elements[0].EndCol != 0 {
		t.Fatalf("expected positions to be applied, got %#v", v.elements[0])
	}
}

func TestViewer_Selected_ReturnsImmutableCopy(t *testing.T) {
	v := New(Options{Renderer: staticRenderer{lines: []string{"x"}}})
	v.elements = []NavElement{{Type: NavElementURL, Text: "original"}}
	v.selectedIndex = 0

	sel := v.Selected()
	if sel == nil {
		t.Fatal("expected non-nil selection")
	}

	// Modifying the returned pointer should NOT affect internal state
	sel.Text = "modified"
	if v.elements[0].Text != "original" {
		t.Fatalf("expected element to remain unchanged; got %q, want %q", v.elements[0].Text, "original")
	}
}

func TestViewer_SetMarkdownWithSource_ErrorDoesNotCorruptState(t *testing.T) {
	// First, set up valid initial state
	v := New(Options{Renderer: staticRenderer{lines: []string{"initial content"}}})
	err := v.SetMarkdown("# Initial")
	if err != nil {
		t.Fatalf("unexpected error on initial SetMarkdown: %v", err)
	}

	initialMarkdown := v.Markdown()
	initialLines := v.RenderedLines()
	if len(initialLines) == 0 {
		t.Fatal("expected initial content to be set")
	}

	// Now switch to an error-producing renderer and try to set new content
	v.SetRenderer(errorRenderer{err: errRenderFailed})
	err = v.SetMarkdownWithSource("# New Content", "new.md", false)

	// Should return error
	if err == nil {
		t.Fatal("expected error from failing renderer")
	}
	if !errors.Is(err, errRenderFailed) {
		t.Fatalf("expected errRenderFailed, got: %v", err)
	}

	// Critical: state should be UNCHANGED after error
	if v.Markdown() != initialMarkdown {
		t.Fatalf("markdown was corrupted after error; got %q, want %q", v.Markdown(), initialMarkdown)
	}
	if len(v.RenderedLines()) != len(initialLines) {
		t.Fatalf("rendered lines were corrupted after error; got %d lines, want %d", len(v.RenderedLines()), len(initialLines))
	}
}

func TestViewer_GoBackDoesNotMutateHistoryWhenUnavailable(t *testing.T) {
	v := New(Options{Renderer: staticRenderer{lines: []string{"x"}}})

	// No history: GoBack should fail and not create forward history.
	if v.GoBack() {
		t.Fatal("expected GoBack to return false when no back history exists")
	}
	if v.history.ForwardStackSize() != 0 {
		t.Fatalf("expected forward history to remain empty, got %d", v.history.ForwardStackSize())
	}
	if v.history.BackStackSize() != 0 {
		t.Fatalf("expected back history to remain empty, got %d", v.history.BackStackSize())
	}
}

func TestViewer_GoForwardDoesNotMutateHistoryWhenUnavailable(t *testing.T) {
	v := New(Options{Renderer: staticRenderer{lines: []string{"x"}}})

	// No history: GoForward should fail and not create back history.
	if v.GoForward() {
		t.Fatal("expected GoForward to return false when no forward history exists")
	}
	if v.history.BackStackSize() != 0 {
		t.Fatalf("expected back history to remain empty, got %d", v.history.BackStackSize())
	}
	if v.history.ForwardStackSize() != 0 {
		t.Fatalf("expected forward history to remain empty, got %d", v.history.ForwardStackSize())
	}
}

func TestViewer_EnsureVisibleNeverProducesNegativeScrollOffset(t *testing.T) {
	v := New(Options{Renderer: staticRenderer{lines: []string{"x"}}})
	v.elements = []NavElement{{Type: NavElementURL, Text: "x", StartLine: 0, EndLine: 0, StartCol: 0, EndCol: 1}}
	v.selectedIndex = 0
	v.scrollOffset = 0

	// Large viewport height; element at the top should never yield a negative offset.
	v.ensureVisible(100)
	if v.scrollOffset < 0 {
		t.Fatalf("expected non-negative scrollOffset, got %d", v.scrollOffset)
	}
}

func TestViewer_MoveToNextLink_NoSelection_DoesNotSelectBelowViewport(t *testing.T) {
	v := New(Options{Renderer: staticRenderer{lines: []string{"x"}}})

	v.scrollOffset = 5
	viewportHeight := 3 // visible lines: 5,6,7

	v.elements = []NavElement{
		{Type: NavElementURL, Text: "above", StartLine: 4, EndLine: 4, StartCol: 0, EndCol: 4},
		{Type: NavElementURL, Text: "below", StartLine: 99, EndLine: 99, StartCol: 0, EndCol: 5},
	}
	v.selectedIndex = -1

	if v.MoveToNextLink(viewportHeight) {
		t.Fatalf("expected MoveToNextLink to return false (no in-viewport links), got selected=%#v", v.Selected())
	}
	if v.selectedIndex != -1 {
		t.Fatalf("expected selection to remain -1, got %d", v.selectedIndex)
	}
}

func TestViewer_MoveToPreviousLink_NoSelection_DoesNotSelectAboveViewport(t *testing.T) {
	v := New(Options{Renderer: staticRenderer{lines: []string{"x"}}})

	v.scrollOffset = 5
	viewportHeight := 3 // visible lines: 5,6,7

	v.elements = []NavElement{
		{Type: NavElementURL, Text: "above", StartLine: 4, EndLine: 4, StartCol: 0, EndCol: 4},
		{Type: NavElementURL, Text: "below", StartLine: 99, EndLine: 99, StartCol: 0, EndCol: 5},
	}
	v.selectedIndex = -1

	if v.MoveToPreviousLink(viewportHeight) {
		t.Fatalf("expected MoveToPreviousLink to return false (no in-viewport links), got selected=%#v", v.Selected())
	}
	if v.selectedIndex != -1 {
		t.Fatalf("expected selection to remain -1, got %d", v.selectedIndex)
	}
}

func TestViewer_RestoreState_OutOfBoundsSelectionCleared(t *testing.T) {
	v := New(Options{Renderer: staticRenderer{lines: []string{"x"}}})
	v.elements = []NavElement{{Type: NavElementURL, Text: "x"}}
	v.selectedIndex = 0

	st := v.saveCurrentState()
	st.SelectedIndex = 12345 // out of bounds for restored elements

	v.restoreState(st)
	if v.selectedIndex != -1 {
		t.Fatalf("expected out-of-bounds selection to be cleared, got %d", v.selectedIndex)
	}
}
