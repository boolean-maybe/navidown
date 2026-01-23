package navidown

import (
	"errors"
	"fmt"
)

// Sentinel errors for ContentFetcher operations.
var (
	// ErrNilViewer is returned when OnSelect is called with a nil viewer.
	ErrNilViewer = errors.New("nil viewer")
	// ErrNotLink is returned when the element is not a URL type.
	ErrNotLink = errors.New("element is not a URL")
	// ErrEmptyContent is returned when the fetched content is empty.
	ErrEmptyContent = errors.New("content is empty")
)

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
// Returns an error if the operation fails. Use OnSelectWithErrorDisplay for backward-compatible
// behavior that shows errors in the viewer.
func (cf *ContentFetcher) OnSelect(viewer *MarkdownSession, elem NavElement) error {
	if viewer == nil {
		return ErrNilViewer
	}
	if elem.Type != NavElementURL {
		return ErrNotLink
	}

	content, err := cf.provider.FetchContent(elem)
	if err != nil {
		return fmt.Errorf("fetch content for %q: %w", elem.URL, err)
	}

	if content == "" {
		return ErrEmptyContent
	}

	newSourcePath := elem.URL
	if !isHTTPURL(elem.URL) && elem.SourceFilePath != "" {
		resolved, rerr := ResolveMarkdownPath(elem.URL, elem.SourceFilePath, cf.searchRoots)
		if rerr == nil && resolved != "" {
			newSourcePath = resolved
		}
	}

	return viewer.SetMarkdownWithSource(content, newSourcePath, true)
}

// OnSelectWithErrorDisplay loads content, showing errors in the viewer.
// This provides backward-compatible behavior for callers that don't handle errors explicitly.
func (cf *ContentFetcher) OnSelectWithErrorDisplay(viewer *MarkdownSession, elem NavElement) {
	if err := cf.OnSelect(viewer, elem); err != nil && !errors.Is(err, ErrNotLink) {
		errorContent := "# Error\n\nFailed to load `" + elem.URL + "`:\n\n```\n" + err.Error() + "\n```"
		_ = viewer.SetMarkdownWithSource(errorContent, elem.SourceFilePath, true)
	}
}
