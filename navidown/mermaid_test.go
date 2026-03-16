package navidown

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newTestMermaidRenderer creates a MermaidRenderer that uses a fake mmdc script
// which copies a 1x1 PNG fixture to the output path.
func newTestMermaidRenderer(t *testing.T) *MermaidRenderer {
	t.Helper()

	// create a fixture 1x1 PNG (minimal valid PNG)
	fixturePNG := minimalPNG()

	// create a fake mmdc script that writes the fixture PNG to the -o argument
	scriptDir := t.TempDir()
	scriptPath := filepath.Join(scriptDir, "fake-mmdc")

	script := fmt.Sprintf(`#!/bin/sh
# parse -o flag to find output path
while [ $# -gt 0 ]; do
  case "$1" in
    -o) shift; cp "%s" "$1" ;;
  esac
  shift
done
`, filepath.Join(scriptDir, "fixture.png"))

	if err := os.WriteFile(filepath.Join(scriptDir, "fixture.png"), fixturePNG, 0644); err != nil {
		t.Fatalf("write fixture PNG: %v", err)
	}
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("write fake mmdc: %v", err)
	}

	renderer := NewMermaidRenderer(MermaidOptions{MmdcPath: scriptPath})
	if renderer == nil {
		t.Fatal("NewMermaidRenderer returned nil")
	}
	t.Cleanup(renderer.Close)
	return renderer
}

// minimalPNG returns bytes of a valid 1x1 white PNG.
func minimalPNG() []byte {
	return []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, // PNG signature
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xde, 0x00, 0x00, 0x00, 0x0c, 0x49, 0x44, 0x41, // IDAT chunk
		0x54, 0x08, 0xd7, 0x63, 0xf8, 0xcf, 0xc0, 0x00,
		0x00, 0x00, 0x02, 0x00, 0x01, 0xe2, 0x21, 0xbc,
		0x33, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, // IEND chunk
		0x44, 0xae, 0x42, 0x60, 0x82,
	}
}

func TestPreprocessMermaid_DetectsBlocks(t *testing.T) {
	renderer := newTestMermaidRenderer(t)

	md := "# Title\n\n```mermaid\ngraph TD\n    A-->B\n```\n\nSome text.\n"
	result := preprocessMermaid(md, renderer)

	if strings.Contains(result, "```mermaid") {
		t.Error("mermaid fence should have been replaced")
	}
	if !strings.Contains(result, "![mermaid diagram](") {
		t.Errorf("expected image syntax in output, got:\n%s", result)
	}
	if !strings.Contains(result, ".png)") {
		t.Error("expected .png path in image syntax")
	}
	// surrounding content preserved
	if !strings.Contains(result, "# Title") {
		t.Error("title should be preserved")
	}
	if !strings.Contains(result, "Some text.") {
		t.Error("trailing text should be preserved")
	}
}

func TestPreprocessMermaid_PreservesNonMermaid(t *testing.T) {
	renderer := newTestMermaidRenderer(t)

	md := "```go\nfunc main() {}\n```\n\n```python\nprint('hello')\n```\n"
	result := preprocessMermaid(md, renderer)

	if result != md {
		t.Errorf("non-mermaid blocks should be untouched.\ngot:  %q\nwant: %q", result, md)
	}
}

func TestPreprocessMermaid_MixedContent(t *testing.T) {
	renderer := newTestMermaidRenderer(t)

	md := "# Intro\n\n```go\nfunc main() {}\n```\n\n```mermaid\nsequenceDiagram\n    A->>B: Hello\n```\n\nEnd.\n"
	result := preprocessMermaid(md, renderer)

	// go block preserved
	if !strings.Contains(result, "```go") {
		t.Error("go block should be preserved")
	}
	// mermaid block replaced
	if strings.Contains(result, "```mermaid") {
		t.Error("mermaid block should have been replaced")
	}
	if !strings.Contains(result, "![mermaid diagram](") {
		t.Error("expected image syntax for mermaid block")
	}
}

func TestPreprocessMermaid_ErrorPreservesBlock(t *testing.T) {
	// create a renderer with a failing mmdc
	scriptDir := t.TempDir()
	scriptPath := filepath.Join(scriptDir, "bad-mmdc")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/sh\nexit 1\n"), 0755); err != nil {
		t.Fatalf("write bad mmdc: %v", err)
	}

	renderer := NewMermaidRenderer(MermaidOptions{MmdcPath: scriptPath})
	if renderer == nil {
		t.Fatal("NewMermaidRenderer returned nil")
	}
	t.Cleanup(renderer.Close)

	md := "```mermaid\ngraph TD\n    A-->B\n```\n"
	result := preprocessMermaid(md, renderer)

	if !strings.Contains(result, "```mermaid") {
		t.Error("on error, original block should be preserved")
	}
}

func TestMermaidRenderer_Caching(t *testing.T) {
	renderer := newTestMermaidRenderer(t)

	source := "graph TD\n    A-->B\n"

	path1, err := renderer.RenderToFile(source)
	if err != nil {
		t.Fatalf("first render: %v", err)
	}

	path2, err := renderer.RenderToFile(source)
	if err != nil {
		t.Fatalf("second render: %v", err)
	}

	if path1 != path2 {
		t.Errorf("cache miss: paths differ: %q vs %q", path1, path2)
	}
}

func TestMermaidRenderer_Close(t *testing.T) {
	renderer := NewMermaidRenderer(MermaidOptions{MmdcPath: "/bin/echo"})
	if renderer == nil {
		t.Fatal("NewMermaidRenderer returned nil")
	}

	cacheDir := renderer.cacheDir
	if _, err := os.Stat(cacheDir); err != nil {
		t.Fatalf("cache dir should exist: %v", err)
	}

	renderer.Close()

	if _, err := os.Stat(cacheDir); !os.IsNotExist(err) {
		t.Error("cache dir should be removed after Close()")
	}
}

func TestPreprocessMermaid_NilRenderer(t *testing.T) {
	md := "```mermaid\ngraph TD\n```\n"
	result := preprocessMermaid(md, nil)
	if result != md {
		t.Error("nil renderer should return markdown unchanged")
	}
}

func TestPreprocessMermaid_MultipleMermaidBlocks(t *testing.T) {
	renderer := newTestMermaidRenderer(t)

	md := "```mermaid\ngraph TD\n    A-->B\n```\n\nMiddle text.\n\n```mermaid\nsequenceDiagram\n    A->>B: Hi\n```\n"
	result := preprocessMermaid(md, renderer)

	count := strings.Count(result, "![mermaid diagram](")
	if count != 2 {
		t.Errorf("expected 2 image replacements, got %d", count)
	}
	if strings.Contains(result, "```mermaid") {
		t.Error("all mermaid blocks should have been replaced")
	}
	if !strings.Contains(result, "Middle text.") {
		t.Error("middle text should be preserved")
	}
}

func TestPreprocessMermaid_FourBacktickFence(t *testing.T) {
	renderer := newTestMermaidRenderer(t)

	md := "````mermaid\ngraph TD\n    A-->B\n````\n"
	result := preprocessMermaid(md, renderer)

	if strings.Contains(result, "````mermaid") {
		t.Error("4-backtick mermaid fence should have been replaced")
	}
	if !strings.Contains(result, "![mermaid diagram](") {
		t.Error("expected image syntax")
	}
}

func TestMarkdownSession_MermaidRendersAsImage(t *testing.T) {
	renderer := newTestMermaidRenderer(t)

	session := New(Options{
		MermaidOptions: &MermaidOptions{MmdcPath: renderer.mmdcPath},
	})
	// replace the auto-created mermaid renderer with our test one
	session.mermaidRenderer = renderer

	md := "# Title\n\n```mermaid\ngraph TD\n    A-->B\n```\n\nSome text.\n"
	if err := session.SetMarkdown(md); err != nil {
		t.Fatalf("SetMarkdown: %v", err)
	}

	// should have image element from mermaid diagram
	var images int
	for _, elem := range session.Elements() {
		if elem.Type == NavElementImage {
			images++
			if elem.Text != "mermaid diagram" {
				t.Errorf("image text: got %q, want %q", elem.Text, "mermaid diagram")
			}
			if !strings.HasSuffix(elem.URL, ".png") {
				t.Errorf("image URL should end with .png: %q", elem.URL)
			}
		}
	}
	if images != 1 {
		t.Errorf("expected 1 image element, got %d", images)
	}

	// rendered output should contain image fallback (no ImagePostProcessor set)
	joined := strings.Join(session.RenderedLines(), "\n")
	if !strings.Contains(joined, "[image: mermaid diagram]") {
		t.Errorf("expected fallback text in output, got:\n%s", joined)
	}
}
