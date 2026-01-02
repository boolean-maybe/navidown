package navidown

// ContentProvider defines the interface for fetching content based on a navigation element.
type ContentProvider interface {
	FetchContent(elem NavElement) (string, error)
}

// ContentFetcher handles selection events by using a ContentProvider to retrieve and update markdown content.
// It is UI-agnostic; a host UI should call OnSelect when the user activates a link.
type ContentFetcher struct {
	provider    ContentProvider
	searchRoots []string
}

// NewContentFetcher creates a new ContentFetcher with the specific provider implementation.
func NewContentFetcher(provider ContentProvider, searchRoots []string) *ContentFetcher {
	return &ContentFetcher{
		provider:    provider,
		searchRoots: searchRoots,
	}
}

// OnSelect loads linked markdown and replaces current content, pushing history when navigating.
func (cf *ContentFetcher) OnSelect(viewer *Viewer, elem NavElement) {
	if viewer == nil {
		return
	}
	if elem.Type != NavElementURL {
		return
	}

	content, err := cf.provider.FetchContent(elem)
	if err != nil {
		errorContent := "# Error\n\nFailed to load `" + elem.URL + "`:\n\n```\n" + err.Error() + "\n```"
		_ = viewer.SetMarkdownWithSource(errorContent, elem.SourceFilePath, true)
		return
	}

	if content == "" {
		return
	}

	newSourcePath := elem.URL
	if !isHTTPURL(elem.URL) && elem.SourceFilePath != "" {
		resolved, rerr := ResolveMarkdownPath(elem.URL, elem.SourceFilePath, cf.searchRoots)
		if rerr == nil && resolved != "" {
			newSourcePath = resolved
		}
	}

	_ = viewer.SetMarkdownWithSource(content, newSourcePath, true)
}
