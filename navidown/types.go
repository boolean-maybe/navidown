package navidown

// NavElementType distinguishes between navigable element types.
type NavElementType int

const (
	NavElementHeader NavElementType = iota
	NavElementURL
)

// NavElement represents a navigable item (header or URL).
//
// Positions are in rendered output coordinates:
// - StartLine/EndLine are 0-indexed line numbers
// - StartCol/EndCol are 0-indexed rune columns in the cleaned (non-decorated) line
type NavElement struct {
	Type           NavElementType
	Text           string // visible text (header text or link text)
	URL            string // for links, the URL; empty for headers
	Level          int    // for headers: 1-6; for links: 0
	SourceFilePath string // path to the markdown file containing this element

	StartLine int
	EndLine   int
	StartCol  int
	EndCol    int
}

// PageState captures the full state of a markdown page for navigation history.
type PageState struct {
	Markdown       string
	SourceFilePath string
	SelectedIndex  int
	ScrollOffset   int
	Elements       []NavElement
	RenderedLines  []string
	Cleaner        LineCleaner
}
