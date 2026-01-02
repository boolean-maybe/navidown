package navidown

import "testing"

func TestScoringCorrelator_WordBoundaryMatching(t *testing.T) {
	correlator := NewScoringCorrelator()

	renderedLines := []string{
		"TikiView appears first.",
		"",
		"## View",
		"",
		"More TikiView text.",
	}

	elem := &NavElement{Type: NavElementHeader, Text: "View", Level: 2}
	lineIdx, _, _, found := correlator.CorrelatePosition(elem, renderedLines, LineCleanerFunc(func(s string) string { return s }))
	if !found {
		t.Fatal("expected to find match")
	}
	if lineIdx != 2 {
		t.Errorf("expected 'View' to match line 2, got %d", lineIdx)
	}
}

func TestScoringCorrelator_URLNotMatchingOtherLinkURL(t *testing.T) {
	correlator := NewScoringCorrelator()

	renderedLines := []string{
		"",                                    // 0
		"## 3. Links",                         // 1
		"",                                    // 2
		"Inline link https://raw.github...",   // 3
		"",                                    // 4
		"Link with title https://example.com", // 5
		"",                                    // 6
		"Reference link https://example.com",  // 7
		"",                                    // 8
		"Auto-link: https://example.com",      // 9
		"",                                    // 10
	}

	autoLinkElem := &NavElement{
		Type: NavElementURL,
		Text: "https://example.com",
		URL:  "https://example.com",
	}

	lineIdx, _, _, found := correlator.CorrelatePosition(autoLinkElem, renderedLines, LineCleanerFunc(func(s string) string { return s }))
	if !found {
		t.Fatal("expected to find auto-link position")
	}
	if lineIdx != 9 {
		t.Errorf("auto-link should match line 9, got line %d", lineIdx)
	}
}
