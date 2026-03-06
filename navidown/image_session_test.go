package navidown

import (
	"strings"
	"testing"
)

func TestMarkdownSession_ImageExtraction(t *testing.T) {
	session := New(Options{})

	md := "# Title\n\n![diagram](arch.png)\n\nSome text with a [link](other.md).\n"
	if err := session.SetMarkdown(md); err != nil {
		t.Fatalf("SetMarkdown: %v", err)
	}

	elements := session.Elements()

	// Should have: 1 header + 1 image + 1 link = 3 elements
	var headers, images, links int
	for _, elem := range elements {
		switch elem.Type {
		case NavElementHeader:
			headers++
		case NavElementImage:
			images++
			if elem.Text != "diagram" {
				t.Errorf("image text: got %q, want %q", elem.Text, "diagram")
			}
			if elem.URL != "arch.png" {
				t.Errorf("image URL: got %q, want %q", elem.URL, "arch.png")
			}
		case NavElementURL:
			links++
		}
	}

	if headers != 1 {
		t.Errorf("headers: got %d, want 1", headers)
	}
	if images != 1 {
		t.Errorf("images: got %d, want 1", images)
	}
	if links != 1 {
		t.Errorf("links: got %d, want 1", links)
	}
}

func TestMarkdownSession_ImageFallback(t *testing.T) {
	// Without an ImagePostProcessor, image tokens should be replaced with fallback text
	session := New(Options{})

	md := "![diagram](test.png)\n"
	if err := session.SetMarkdown(md); err != nil {
		t.Fatalf("SetMarkdown: %v", err)
	}

	lines := session.RenderedLines()
	joined := strings.Join(lines, "\n")

	// The fallback should show "[image: diagram]" text
	if strings.Contains(joined, "\uFFF0") || strings.Contains(joined, "\uFFF1") {
		t.Error("rendered output should not contain raw image tokens")
	}
	if !strings.Contains(joined, "[image: diagram]") {
		t.Errorf("expected fallback text '[image: diagram]' in output, got: %q", joined)
	}
}

func TestMarkdownSession_ImagePostProcessor(t *testing.T) {
	// Custom processor that replaces image tokens with custom text
	processor := &testImageProcessor{}
	session := New(Options{
		ImagePostProcessor: processor,
	})

	md := "![photo](pic.jpg)\n"
	if err := session.SetMarkdown(md); err != nil {
		t.Fatalf("SetMarkdown: %v", err)
	}

	lines := session.RenderedLines()
	joined := strings.Join(lines, "\n")

	if !strings.Contains(joined, "[TEST:pic.jpg]") {
		t.Errorf("expected custom processor output, got: %q", joined)
	}
}

type testImageProcessor struct{}

func (p *testImageProcessor) ProcessImageTokens(lines []string, _ string, _ int) []string {
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		if !ContainsImageToken(line) {
			result = append(result, line)
			continue
		}
		replaced := replaceImageTokensInLine(line, func(url, _ string) string {
			return "[TEST:" + url + "]"
		})
		result = append(result, replaced)
	}
	return result
}
