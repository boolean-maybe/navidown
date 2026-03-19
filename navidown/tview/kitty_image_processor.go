package tview

import (
	"strings"

	nav "github.com/boolean-maybe/navidown/navidown"
)

// KittyImageProcessor replaces image placeholder tokens with Kitty Unicode
// placeholder lines. It uses an ImageManager to resolve images and allocate IDs.
type KittyImageProcessor struct {
	manager *ImageManager
}

// NewKittyImageProcessor creates a new processor backed by the given manager.
func NewKittyImageProcessor(manager *ImageManager) *KittyImageProcessor {
	return &KittyImageProcessor{manager: manager}
}

// ProcessImageTokens scans lines for image tokens and replaces them with
// Kitty Unicode placeholder rows. Images are pre-resolved in parallel before
// sequential placeholder generation.
func (p *KittyImageProcessor) ProcessImageTokens(lines []string, sourceFilePath string, maxCols int) []string {
	if maxCols <= 0 {
		maxCols = 80
	}

	// first pass: collect all image URLs for parallel pre-resolution
	urls := p.collectImageURLs(lines)
	if len(urls) > 0 {
		p.manager.PreResolveImages(urls, sourceFilePath)
	}

	// second pass: build output lines (resolver cache is warm)
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		if !nav.ContainsImageToken(line) {
			result = append(result, line)
			continue
		}

		expanded := p.processLine(line, sourceFilePath, maxCols)
		result = append(result, expanded...)
	}
	return result
}

// collectImageURLs extracts all image URLs from token-bearing lines.
func (p *KittyImageProcessor) collectImageURLs(lines []string) []string {
	var urls []string
	seen := make(map[string]bool)
	for _, line := range lines {
		if !nav.ContainsImageToken(line) {
			continue
		}
		remaining := line
		for {
			startIdx := findTokenStart(remaining)
			if startIdx < 0 {
				break
			}
			endIdx := findTokenEnd(remaining[startIdx:])
			if endIdx < 0 {
				break
			}
			endIdx += startIdx
			token := remaining[startIdx:endIdx]
			if url, _, ok := nav.ParseImageToken(token); ok && !seen[url] {
				seen[url] = true
				urls = append(urls, url)
			}
			remaining = remaining[endIdx:]
		}
	}
	return urls
}

func (p *KittyImageProcessor) processLine(line string, sourceFilePath string, maxCols int) []string {
	// A line may contain text before/after the image token.
	// For simplicity, if the line contains an image token, we replace the
	// token with placeholder lines and keep surrounding text on separate lines.
	var result []string

	remaining := line
	for {
		startIdx := findTokenStart(remaining)
		if startIdx < 0 {
			if remaining != "" {
				result = append(result, remaining)
			}
			break
		}

		// Emit text before the token
		if startIdx > 0 {
			result = append(result, remaining[:startIdx])
		}

		endIdx := findTokenEnd(remaining[startIdx:])
		if endIdx < 0 {
			result = append(result, remaining[startIdx:])
			break
		}
		endIdx += startIdx

		token := remaining[startIdx:endIdx]
		url, alt, ok := nav.ParseImageToken(token)
		if !ok {
			result = append(result, token)
			remaining = remaining[endIdx:]
			continue
		}

		// Try to resolve and generate placeholders
		placeholder, err := p.manager.ResolveAndAllocate(url, sourceFilePath, maxCols)
		if err != nil {
			// Fallback: show alt text
			if alt != "" {
				result = append(result, "[image: "+alt+"]")
			} else {
				result = append(result, "[image: "+url+"]")
			}
			remaining = remaining[endIdx:]
			continue
		}

		// Generate placeholder lines
		placeholderLines := BuildPlaceholderLines(placeholder)
		result = append(result, placeholderLines...)

		remaining = remaining[endIdx:]
	}

	if len(result) == 0 {
		return []string{""}
	}
	return result
}

func findTokenStart(s string) int {
	return strings.Index(s, nav.ImageTokenStart+"IMG:")
}

func findTokenEnd(s string) int {
	idx := strings.Index(s, nav.ImageTokenEnd)
	if idx < 0 {
		return -1
	}
	return idx + len(nav.ImageTokenEnd)
}
