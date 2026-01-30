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
	if sel := v.Selected(); sel == nil || sel.Text != "Link 1" {
		t.Fatalf("expected Link 1 selected, got %#v", sel)
	}

	if !v.MoveToNextLink(10) {
		t.Fatal("expected to move to second link")
	}
	if sel := v.Selected(); sel == nil || sel.Text != "Link 2" {
		t.Fatalf("expected Link 2 selected, got %#v", sel)
	}
}

func TestViewer_HistoryPreservesSelection(t *testing.T) {
	r1 := staticRenderer{lines: []string{"Link A", "Link B", "Link C"}}
	v := New(Options{Renderer: r1})

	_ = v.SetMarkdownWithSource("# Page 1\n[Link A](a.md)\n[Link B](b.md)\n[Link C](c.md)", "page1.md", false)
	v.MoveToNextLink(10) // A
	v.MoveToNextLink(10) // B
	v.MoveToNextLink(10) // C
	if sel := v.Selected(); sel == nil || sel.Text != "Link C" {
		t.Fatalf("expected Link C selected, got %#v", sel)
	}

	// Swap renderer for page 2 so correlation still yields non-zero spans.
	v.SetRenderer(staticRenderer{lines: []string{"Link X"}})
	_ = v.SetMarkdownWithSource("# Page 2\n[Link X](x.md)", "page2.md", true)
	if !v.GoBack() {
		t.Fatal("expected GoBack to succeed")
	}

	if sel := v.Selected(); sel == nil || sel.Text != "Link C" {
		t.Fatalf("expected Link C restored, got %#v", sel)
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
	if sel := v.Selected(); sel == nil || sel.Text != "good" {
		t.Fatalf("expected 'good' selected, got %#v", sel)
	}

	// MoveToLast should also pick the last valid link, not the degenerate one.
	v.selectedIndex = -1
	if !v.MoveToLast(10) {
		t.Fatal("expected MoveToLast to succeed")
	}
	if sel := v.Selected(); sel == nil || sel.Text != "good" {
		t.Fatalf("expected 'good' selected, got %#v", sel)
	}

	// Backward navigation should skip the degenerate one.
	v.selectedIndex = 1 // pretend we were on the degenerate one
	if !v.MoveToPreviousLink(10) {
		t.Fatal("expected MoveToPreviousLink to succeed")
	}
	if sel := v.Selected(); sel == nil || sel.Text != "good" {
		t.Fatalf("expected 'good' selected, got %#v", sel)
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

func TestViewer_VisibleLines_ClampsAndHandlesEmpty(t *testing.T) {
	v := New(Options{Renderer: staticRenderer{lines: []string{"a", "b", "c"}}})
	_ = v.SetMarkdown("a\nb\nc")

	if got := v.VisibleLines(0); got != nil {
		t.Fatalf("expected nil when viewportHeight <= 0, got %#v", got)
	}

	v.scrollOffset = -5
	got := v.VisibleLines(2)
	if v.scrollOffset != 0 {
		t.Fatalf("expected scrollOffset clamped to 0, got %d", v.scrollOffset)
	}
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("unexpected visible lines: %#v", got)
	}

	v.scrollOffset = 99
	if got = v.VisibleLines(2); got != nil {
		t.Fatalf("expected nil when scrollOffset past end, got %#v", got)
	}
}

func TestViewer_ScrollUpDown_ClearsSelectionOffscreen(t *testing.T) {
	v := New(Options{Renderer: staticRenderer{lines: []string{"0", "1", "2", "3", "4"}}})
	_ = v.SetMarkdown("0\n1\n2\n3\n4")
	v.elements = []NavElement{{Type: NavElementURL, Text: "x", StartLine: 4, EndLine: 4, StartCol: 0, EndCol: 1}}
	v.selectedIndex = 0

	if v.ScrollUp(1) {
		t.Fatal("expected ScrollUp at top to return false")
	}

	if !v.ScrollDown(1) {
		t.Fatal("expected ScrollDown to return true")
	}
	if v.selectedIndex != -1 {
		t.Fatalf("expected selection cleared when offscreen, got %d", v.selectedIndex)
	}
}

func TestViewer_PageHomeEnd_Offsets(t *testing.T) {
	v := New(Options{Renderer: staticRenderer{lines: []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}}})
	_ = v.SetMarkdown("0\n1\n2\n3\n4\n5\n6\n7\n8\n9")

	if !v.PageDown(3) {
		t.Fatal("expected PageDown to return true")
	}
	if v.scrollOffset != 3 {
		t.Fatalf("expected scrollOffset=3 after PageDown, got %d", v.scrollOffset)
	}

	if !v.PageUp(3) {
		t.Fatal("expected PageUp to return true")
	}
	if v.scrollOffset != 0 {
		t.Fatalf("expected scrollOffset=0 after PageUp, got %d", v.scrollOffset)
	}

	v.End(3)
	if v.scrollOffset != 7 {
		t.Fatalf("expected scrollOffset=7 after End, got %d", v.scrollOffset)
	}

	v.Home(3)
	if v.scrollOffset != 0 {
		t.Fatalf("expected scrollOffset=0 after Home, got %d", v.scrollOffset)
	}
}

func TestViewer_HistoryMaxEvictsOldest(t *testing.T) {
	v := New(Options{Renderer: staticRenderer{lines: []string{"x"}}, HistoryMax: 1})

	_ = v.SetMarkdownWithSource("# page1", "page1.md", false)
	_ = v.SetMarkdownWithSource("# page2", "page2.md", true)
	_ = v.SetMarkdownWithSource("# page3", "page3.md", true)

	if !v.GoBack() {
		t.Fatal("expected GoBack to succeed")
	}
	if v.SourceFilePath() != "page2.md" {
		t.Fatalf("expected page2.md after GoBack, got %q", v.SourceFilePath())
	}
	if v.GoBack() {
		t.Fatal("expected GoBack to fail after eviction")
	}
}

func TestViewer_ParsesAutoLink(t *testing.T) {
	v := New(Options{Renderer: staticRenderer{lines: []string{"x"}}})
	_ = v.SetMarkdown("See <https://example.com> for details.")

	var found bool
	for _, elem := range v.Elements() {
		if elem.Type == NavElementURL && elem.URL == "https://example.com" && elem.Text == "https://example.com" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected autolink element to be parsed")
	}
}

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World", "hello-world"},
		{"What's New?", "whats-new"},
		{"Version 2.0", "version-20"},
		{"  Spaces  ", "spaces"},
		{"UPPERCASE", "uppercase"},
		{"multiple---hyphens", "multiple-hyphens"},
		{"under_scores_work", "under-scores-work"},
		{"", ""},
		{"123", "123"},
		{"a", "a"},
		{"Special!@#$%Characters", "specialcharacters"},
		{"  leading and trailing  ", "leading-and-trailing"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := generateSlug(tt.input)
			if got != tt.expected {
				t.Errorf("generateSlug(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestNavElement_IsInternalLink(t *testing.T) {
	tests := []struct {
		name     string
		elem     NavElement
		expected bool
	}{
		{"internal link", NavElement{Type: NavElementURL, URL: "#section"}, true},
		{"external link", NavElement{Type: NavElementURL, URL: "https://example.com"}, false},
		{"local file", NavElement{Type: NavElementURL, URL: "other.md"}, false},
		{"empty URL", NavElement{Type: NavElementURL, URL: ""}, false},
		{"header element", NavElement{Type: NavElementHeader, URL: ""}, false},
		{"hash only", NavElement{Type: NavElementURL, URL: "#"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.elem.IsInternalLink()
			if got != tt.expected {
				t.Errorf("IsInternalLink() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNavElement_AnchorTarget(t *testing.T) {
	tests := []struct {
		name     string
		elem     NavElement
		expected string
	}{
		{"internal link", NavElement{Type: NavElementURL, URL: "#section"}, "section"},
		{"external link", NavElement{Type: NavElementURL, URL: "https://example.com"}, ""},
		{"hash only", NavElement{Type: NavElementURL, URL: "#"}, ""},
		{"complex anchor", NavElement{Type: NavElementURL, URL: "#my-header-123"}, "my-header-123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.elem.AnchorTarget()
			if got != tt.expected {
				t.Errorf("AnchorTarget() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestViewer_FindHeaderBySlug(t *testing.T) {
	v := New(Options{Renderer: staticRenderer{lines: []string{"# First", "## Second", "content"}}})
	_ = v.SetMarkdown("# First\n## Second\nSome content")

	// should find existing header
	header := v.FindHeaderBySlug("first")
	if header == nil {
		t.Fatal("expected to find header 'first'")
	}
	if header.Text != "First" {
		t.Errorf("expected Text='First', got %q", header.Text)
	}

	header = v.FindHeaderBySlug("second")
	if header == nil {
		t.Fatal("expected to find header 'second'")
	}

	// should return nil for non-existent
	header = v.FindHeaderBySlug("nonexistent")
	if header != nil {
		t.Errorf("expected nil for nonexistent slug, got %#v", header)
	}
}

func TestViewer_ScrollToAnchor(t *testing.T) {
	v := New(Options{Renderer: staticRenderer{lines: []string{
		"# First",
		"content",
		"more content",
		"## Second",
		"even more",
	}}})
	_ = v.SetMarkdown("# First\ncontent\nmore content\n## Second\neven more")

	// manually set element positions for test
	v.elements[0].StartLine = 0
	v.elements[1].StartLine = 3

	// scroll to second header
	viewportHeight := 3
	totalLines := len(v.RenderedLines())
	maxOffset := totalLines - viewportHeight
	if maxOffset < 0 {
		maxOffset = 0
	}

	if !v.ScrollToAnchor("second", viewportHeight, false) {
		t.Fatal("expected ScrollToAnchor to succeed")
	}
	// header is at line 3, but should be clamped to maxOffset
	if v.ScrollOffset() > maxOffset {
		t.Errorf("scrollOffset %d exceeds maxOffset %d", v.ScrollOffset(), maxOffset)
	}
	if v.ScrollOffset() < 0 {
		t.Errorf("scrollOffset %d is negative", v.ScrollOffset())
	}

	// scroll to first header (at line 0)
	if !v.ScrollToAnchor("first", viewportHeight, false) {
		t.Fatal("expected ScrollToAnchor to succeed")
	}
	if v.ScrollOffset() != 0 {
		t.Errorf("expected scrollOffset=0, got %d", v.ScrollOffset())
	}

	// non-existent should fail
	if v.ScrollToAnchor("nonexistent", viewportHeight, false) {
		t.Error("expected ScrollToAnchor to fail for nonexistent slug")
	}
}

func TestViewer_ScrollToAnchor_PushesToHistory(t *testing.T) {
	v := New(Options{Renderer: staticRenderer{lines: []string{
		"# First",
		"content",
		"## Second",
		"more",
	}}})
	_ = v.SetMarkdown("# First\ncontent\n## Second\nmore")

	v.elements[0].StartLine = 0
	v.elements[1].StartLine = 2

	// initial state
	if v.CanGoBack() {
		t.Fatal("expected no back history initially")
	}

	// scroll with pushToHistory=true
	if !v.ScrollToAnchor("second", 4, true) {
		t.Fatal("expected ScrollToAnchor to succeed")
	}

	if !v.CanGoBack() {
		t.Fatal("expected back history after ScrollToAnchor with pushToHistory=true")
	}

	// go back should restore previous position
	if !v.GoBack() {
		t.Fatal("expected GoBack to succeed")
	}
	if v.ScrollOffset() != 0 {
		t.Errorf("expected scrollOffset=0 after GoBack, got %d", v.ScrollOffset())
	}
}

func TestViewer_InternalLinkHistory(t *testing.T) {
	v := New(Options{Renderer: staticRenderer{lines: []string{
		"# Intro",
		"[Jump to Details](#details)",
		"## Details",
		"Some details here",
		"[Back to Intro](#intro)",
	}}})
	_ = v.SetMarkdown("# Intro\n[Jump to Details](#details)\n## Details\nSome details here\n[Back to Intro](#intro)")

	// set positions manually
	for i := range v.elements {
		if v.elements[i].Type == NavElementHeader && v.elements[i].Slug == "intro" {
			v.elements[i].StartLine = 0
		}
		if v.elements[i].Type == NavElementHeader && v.elements[i].Slug == "details" {
			v.elements[i].StartLine = 2
		}
	}

	// navigate to #details
	v.ScrollToAnchor("details", 5, true)
	if v.ScrollOffset() != 0 { // 5 lines, viewport 5 → maxOffset=0
		t.Errorf("expected scrollOffset=0, got %d", v.ScrollOffset())
	}

	// go back
	if !v.GoBack() {
		t.Fatal("expected GoBack to succeed")
	}

	// go forward
	if !v.GoForward() {
		t.Fatal("expected GoForward to succeed")
	}
}

func TestViewer_ParsesHeaderSlugs(t *testing.T) {
	v := New(Options{Renderer: staticRenderer{lines: []string{"x"}}})
	_ = v.SetMarkdown("# Hello World\n## What's New?\n### Version 2.0")

	slugs := map[string]bool{}
	for _, elem := range v.Elements() {
		if elem.Type == NavElementHeader {
			slugs[elem.Slug] = true
		}
	}

	expected := []string{"hello-world", "whats-new", "version-20"}
	for _, s := range expected {
		if !slugs[s] {
			t.Errorf("expected slug %q to be present", s)
		}
	}
}

func TestViewer_DuplicateHeaderSlugs(t *testing.T) {
	v := New(Options{Renderer: staticRenderer{lines: []string{
		"Example",
		"First section",
		"Example",
		"Second section",
		"Example",
		"Third section",
	}}})
	_ = v.SetMarkdown(`## Example
First section
## Example
Second section
## Example
Third section`)

	elements := v.Elements()
	if len(elements) != 3 {
		t.Fatalf("expected 3 headers, got %d", len(elements))
	}

	expectedSlugs := []string{"example", "example-1", "example-2"}
	for i, expected := range expectedSlugs {
		if elements[i].Slug != expected {
			t.Errorf("header %d: expected slug %q, got %q", i, expected, elements[i].Slug)
		}
	}

	// Verify FindHeaderBySlug finds each one
	for _, slug := range expectedSlugs {
		found := v.FindHeaderBySlug(slug)
		if found == nil {
			t.Errorf("FindHeaderBySlug(%q) returned nil", slug)
			continue
		}
		if found.Slug != slug {
			t.Errorf("FindHeaderBySlug(%q) returned header with slug %q", slug, found.Slug)
		}
	}
}
