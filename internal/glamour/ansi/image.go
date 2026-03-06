package ansi

import (
	"io"
	"strings"
)

// Image placeholder token delimiters (must match navidown.ImageTokenStart/End).
const (
	imageTokenStart = "\uFFF0"
	imageTokenEnd   = "\uFFF1"
	imageFieldSep   = "\x00" // Separates URL from alt text
)

// An ImageElement is used to render images elements.
type ImageElement struct {
	Text     string
	BaseURL  string
	URL      string
	Child    ElementRenderer
	TextOnly bool
}

// Render renders an ImageElement.
// When the URL points to an image file, it emits a deferred placeholder token
// that will be replaced with Kitty Unicode placeholders during post-processing.
// The token format is: \uFFF0IMG:<resolved-url>\x00<alt-text>\uFFF1
func (e *ImageElement) Render(w io.Writer, ctx RenderContext) error {
	resolvedURL := e.URL
	if len(e.URL) > 0 && e.BaseURL != "" {
		resolvedURL = resolveRelativeURL(e.BaseURL, e.URL)
	}

	// If the URL looks like an image, emit a placeholder token
	if !e.TextOnly && len(resolvedURL) > 0 && looksLikeImage(resolvedURL) {
		token := imageTokenStart + "IMG:" + resolvedURL + imageFieldSep + e.Text + imageTokenEnd
		_, err := io.WriteString(w, token)
		return err
	}

	// Fallback: render as styled text (original behavior)
	style := ctx.options.Styles.ImageText
	if e.TextOnly {
		style.Format = strings.TrimSuffix(style.Format, " →")
	}

	if len(e.Text) > 0 {
		el := &BaseElement{
			Token: e.Text,
			Style: style,
		}
		if err := el.Render(w, ctx); err != nil {
			return err
		}
	}

	if e.TextOnly {
		return nil
	}

	if len(e.URL) > 0 {
		el := &BaseElement{
			Token:  resolvedURL,
			Prefix: " ",
			Style:  ctx.options.Styles.Image,
		}
		if err := el.Render(w, ctx); err != nil {
			return err
		}
	}

	return nil
}

// looksLikeImage returns true if the URL appears to reference an image file.
func looksLikeImage(url string) bool {
	lower := strings.ToLower(url)
	// Strip query parameters and fragments
	if idx := strings.IndexAny(lower, "?#"); idx >= 0 {
		lower = lower[:idx]
	}
	for _, ext := range []string{".png", ".jpg", ".jpeg", ".gif", ".bmp", ".webp", ".tiff", ".tif"} {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}
