package navidown

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// newTestGraphvizRenderer creates a GraphvizRenderer that uses a fake dot script
// which copies a 1x1 PNG fixture to the output path.
func newTestGraphvizRenderer(t *testing.T) *GraphvizRenderer {
	t.Helper()

	fixturePNG := minimalPNG()
	scriptDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(scriptDir, "fixture.png"), fixturePNG, 0644); err != nil {
		t.Fatalf("write fixture PNG: %v", err)
	}

	scriptPath := writeFakeDot(t, scriptDir, filepath.Join(scriptDir, "fixture.png"))

	cacheDir := t.TempDir()
	renderer := NewGraphvizRenderer(GraphvizOptions{DotPath: scriptPath, CacheDir: cacheDir})
	if renderer == nil {
		t.Fatal("NewGraphvizRenderer returned nil")
	}
	t.Cleanup(renderer.Close)
	return renderer
}

// writeFakeDot creates a platform-appropriate fake dot executable that copies
// fixturePath to the -o argument. Returns the path to the executable.
func writeFakeDot(t *testing.T, dir, fixturePath string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		return writeFakeDotBat(t, dir, fixturePath)
	}
	return writeFakeDotSh(t, dir, fixturePath)
}

func writeFakeDotSh(t *testing.T, dir, fixturePath string) string {
	t.Helper()
	scriptPath := filepath.Join(dir, "fake-dot")
	script := fmt.Sprintf(`#!/bin/sh
while [ $# -gt 0 ]; do
  case "$1" in
    -o) shift; cp "%s" "$1" ;;
  esac
  shift
done
`, fixturePath)
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("write fake dot: %v", err)
	}
	return scriptPath
}

func writeFakeDotBat(t *testing.T, dir, fixturePath string) string {
	t.Helper()
	batPath := filepath.Join(dir, "fake-dot.bat")
	fixturePath = filepath.FromSlash(fixturePath)
	script := fmt.Sprintf("@echo off\r\n:loop\r\nif \"%%~1\"==\"\" goto end\r\nif \"%%~1\"==\"-o\" (\r\n  copy /Y \"%s\" \"%%~2\" >nul\r\n  shift\r\n)\r\nshift\r\ngoto loop\r\n:end\r\n", fixturePath)
	if err := os.WriteFile(batPath, []byte(script), 0755); err != nil {
		t.Fatalf("write fake dot.bat: %v", err)
	}
	return batPath
}

// writeFailingDot creates a platform-appropriate fake dot that always exits with code 1.
func writeFailingDot(t *testing.T, dir string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		p := filepath.Join(dir, "bad-dot.bat")
		if err := os.WriteFile(p, []byte("@echo off\r\nexit /b 1\r\n"), 0755); err != nil {
			t.Fatalf("write bad dot.bat: %v", err)
		}
		return p
	}
	p := filepath.Join(dir, "bad-dot")
	if err := os.WriteFile(p, []byte("#!/bin/sh\nexit 1\n"), 0755); err != nil {
		t.Fatalf("write bad dot: %v", err)
	}
	return p
}

// writeFakeDotWithCounter creates a fake dot that copies fixturePath to the -o
// argument AND increments a counter file on each invocation.
func writeFakeDotWithCounter(t *testing.T, dir, fixturePath string) (string, string) {
	t.Helper()
	counterPath := filepath.Join(dir, "counter")
	if runtime.GOOS == "windows" {
		return writeFakeDotWithCounterBat(t, dir, fixturePath, counterPath), counterPath
	}
	return writeFakeDotWithCounterSh(t, dir, fixturePath, counterPath), counterPath
}

func writeFakeDotWithCounterSh(t *testing.T, dir, fixturePath, counterPath string) string {
	t.Helper()
	scriptPath := filepath.Join(dir, "fake-dot")
	script := fmt.Sprintf(`#!/bin/sh
count=0
if [ -f "%s" ]; then
  count=$(cat "%s")
fi
count=$((count + 1))
echo $count > "%s"
while [ $# -gt 0 ]; do
  case "$1" in
    -o) shift; cp "%s" "$1" ;;
  esac
  shift
done
`, counterPath, counterPath, counterPath, fixturePath)
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("write fake dot: %v", err)
	}
	return scriptPath
}

func writeFakeDotWithCounterBat(t *testing.T, dir, fixturePath, counterPath string) string {
	t.Helper()
	batPath := filepath.Join(dir, "fake-dot.bat")
	fixturePath = filepath.FromSlash(fixturePath)
	counterPath = filepath.FromSlash(counterPath)
	script := fmt.Sprintf("@echo off\r\nsetlocal enabledelayedexpansion\r\nset count=0\r\nif exist \"%s\" (\r\n  set /p count=<\"%s\"\r\n)\r\nset /a count=count+1\r\necho !count!>\"%s\"\r\n:loop\r\nif \"%%~1\"==\"\" goto end\r\nif \"%%~1\"==\"-o\" (\r\n  copy /Y \"%s\" \"%%~2\" >nul\r\n  shift\r\n)\r\nshift\r\ngoto loop\r\n:end\r\n", counterPath, counterPath, counterPath, fixturePath)
	if err := os.WriteFile(batPath, []byte(script), 0755); err != nil {
		t.Fatalf("write fake dot.bat: %v", err)
	}
	return batPath
}

func TestPreprocessGraphviz_DetectsBlocks(t *testing.T) {
	renderer := newTestGraphvizRenderer(t)

	md := "# Title\n\n```dot\ndigraph { A -> B }\n```\n\nSome text.\n"
	result := preprocessGraphviz(md, renderer)

	if strings.Contains(result, "```dot") {
		t.Error("dot fence should have been replaced")
	}
	if !strings.Contains(result, "![dot diagram](") {
		t.Errorf("expected image syntax in output, got:\n%s", result)
	}
	if !strings.Contains(result, ".png)") {
		t.Error("expected .png path in image syntax")
	}
	if !strings.Contains(result, "# Title") {
		t.Error("title should be preserved")
	}
	if !strings.Contains(result, "Some text.") {
		t.Error("trailing text should be preserved")
	}
}

func TestPreprocessGraphviz_GraphvizFenceName(t *testing.T) {
	renderer := newTestGraphvizRenderer(t)

	md := "```graphviz\ndigraph { A -> B }\n```\n"
	result := preprocessGraphviz(md, renderer)

	if strings.Contains(result, "```graphviz") {
		t.Error("graphviz fence should have been replaced")
	}
	if !strings.Contains(result, "![dot diagram](") {
		t.Errorf("expected image syntax in output, got:\n%s", result)
	}
}

func TestPreprocessGraphviz_PreservesNonGraphviz(t *testing.T) {
	renderer := newTestGraphvizRenderer(t)

	md := "```go\nfunc main() {}\n```\n\n```python\nprint('hello')\n```\n"
	result := preprocessGraphviz(md, renderer)

	if result != md {
		t.Errorf("non-graphviz blocks should be untouched.\ngot:  %q\nwant: %q", result, md)
	}
}

func TestPreprocessGraphviz_MixedContent(t *testing.T) {
	renderer := newTestGraphvizRenderer(t)

	md := "# Intro\n\n```go\nfunc main() {}\n```\n\n```dot\ndigraph { A -> B }\n```\n\nEnd.\n"
	result := preprocessGraphviz(md, renderer)

	if !strings.Contains(result, "```go") {
		t.Error("go block should be preserved")
	}
	if strings.Contains(result, "```dot") {
		t.Error("dot block should have been replaced")
	}
	if !strings.Contains(result, "![dot diagram](") {
		t.Error("expected image syntax for dot block")
	}
}

func TestPreprocessGraphviz_ErrorPreservesBlock(t *testing.T) {
	scriptDir := t.TempDir()
	scriptPath := writeFailingDot(t, scriptDir)

	cacheDir := t.TempDir()
	renderer := NewGraphvizRenderer(GraphvizOptions{DotPath: scriptPath, CacheDir: cacheDir})
	if renderer == nil {
		t.Fatal("NewGraphvizRenderer returned nil")
	}
	t.Cleanup(renderer.Close)

	md := "```dot\ndigraph { A -> B }\n```\n"
	result := preprocessGraphviz(md, renderer)

	if !strings.Contains(result, "```dot") {
		t.Error("on error, original block should be preserved")
	}
}

func TestPreprocessGraphviz_NilRenderer(t *testing.T) {
	md := "```dot\ndigraph { A -> B }\n```\n"
	result := preprocessGraphviz(md, nil)
	if result != md {
		t.Error("nil renderer should return markdown unchanged")
	}
}

func TestPreprocessGraphviz_MultipleBlocks(t *testing.T) {
	renderer := newTestGraphvizRenderer(t)

	md := "```dot\ndigraph { A -> B }\n```\n\nMiddle text.\n\n```graphviz\ndigraph { C -> D }\n```\n"
	result := preprocessGraphviz(md, renderer)

	count := strings.Count(result, "![dot diagram](")
	if count != 2 {
		t.Errorf("expected 2 image replacements, got %d", count)
	}
	if strings.Contains(result, "```dot") || strings.Contains(result, "```graphviz") {
		t.Error("all dot/graphviz blocks should have been replaced")
	}
	if !strings.Contains(result, "Middle text.") {
		t.Error("middle text should be preserved")
	}
}

func TestPreprocessGraphviz_FourBacktickFence(t *testing.T) {
	renderer := newTestGraphvizRenderer(t)

	md := "````dot\ndigraph { A -> B }\n````\n"
	result := preprocessGraphviz(md, renderer)

	if strings.Contains(result, "````dot") {
		t.Error("4-backtick dot fence should have been replaced")
	}
	if !strings.Contains(result, "![dot diagram](") {
		t.Error("expected image syntax")
	}
}

func TestPreprocessGraphviz_UnclosedFence(t *testing.T) {
	renderer := newTestGraphvizRenderer(t)

	md := "```dot\ndigraph { A -> B }\n"
	result := preprocessGraphviz(md, renderer)

	if result != md {
		t.Errorf("unclosed fence should be left as-is.\ngot:  %q\nwant: %q", result, md)
	}
}

func TestGraphvizRenderer_Caching(t *testing.T) {
	renderer := newTestGraphvizRenderer(t)

	source := "digraph { A -> B }\n"

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

func TestGraphvizRenderer_DiskCache(t *testing.T) {
	fixturePNG := minimalPNG()
	scriptDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(scriptDir, "fixture.png"), fixturePNG, 0644); err != nil {
		t.Fatal(err)
	}
	scriptPath, counterPath := writeFakeDotWithCounter(t, scriptDir, filepath.Join(scriptDir, "fixture.png"))

	cacheDir := t.TempDir()
	source := "digraph { A -> B }\n"

	// first renderer: renders and writes to disk cache
	r1 := NewGraphvizRenderer(GraphvizOptions{DotPath: scriptPath, CacheDir: cacheDir})
	if r1 == nil {
		t.Fatal("r1 is nil")
	}
	path1, err := r1.RenderToFile(source)
	if err != nil {
		t.Fatalf("r1 render: %v", err)
	}
	r1.Close()

	// second renderer: should find PNG on disk, no dot call
	r2 := NewGraphvizRenderer(GraphvizOptions{DotPath: scriptPath, CacheDir: cacheDir})
	if r2 == nil {
		t.Fatal("r2 is nil")
	}
	path2, err := r2.RenderToFile(source)
	if err != nil {
		t.Fatalf("r2 render: %v", err)
	}
	r2.Close()

	if path1 != path2 {
		t.Errorf("paths differ: %q vs %q", path1, path2)
	}

	// counter should be 1 — dot was only called once
	data, err := os.ReadFile(counterPath)
	if err != nil {
		t.Fatalf("read counter: %v", err)
	}
	if strings.TrimSpace(string(data)) != "1" {
		t.Errorf("dot was called %s times, expected 1", strings.TrimSpace(string(data)))
	}
}

func TestGraphvizRenderer_CacheKeyIncludesOptions(t *testing.T) {
	r1 := &GraphvizRenderer{opts: GraphvizOptions{Layout: "dot"}}
	r2 := &GraphvizRenderer{opts: GraphvizOptions{Layout: "neato"}}

	source := "digraph { A -> B }\n"
	k1 := r1.cacheKey(source)
	k2 := r2.cacheKey(source)

	if k1 == k2 {
		t.Error("different layouts should produce different cache keys")
	}

	r3 := &GraphvizRenderer{opts: GraphvizOptions{Layout: "dot"}}
	k3 := r3.cacheKey(source)
	if k1 != k3 {
		t.Error("same options should produce same cache key")
	}

	// DPI difference
	r4 := &GraphvizRenderer{opts: GraphvizOptions{DPI: 72}}
	r5 := &GraphvizRenderer{opts: GraphvizOptions{DPI: 144}}
	k4 := r4.cacheKey(source)
	k5 := r5.cacheKey(source)
	if k4 == k5 {
		t.Error("different DPI should produce different cache keys")
	}
}

func TestGraphvizRenderer_Close_PersistentDir(t *testing.T) {
	persistentDir := t.TempDir()
	dummyExe := dummyExecutable()

	renderer := NewGraphvizRenderer(GraphvizOptions{
		DotPath:  dummyExe,
		CacheDir: persistentDir,
	})
	if renderer == nil {
		t.Fatal("NewGraphvizRenderer returned nil")
	}

	renderer.Close()

	if _, err := os.Stat(persistentDir); err != nil {
		t.Error("persistent cache dir should survive Close()")
	}
}

func TestGraphvizRenderer_Close_TempDir(t *testing.T) {
	renderer := &GraphvizRenderer{
		opts:    GraphvizOptions{},
		dotPath: dummyExecutable(),
	}

	td, err := os.MkdirTemp("", "navidown-graphviz-test-")
	if err != nil {
		t.Fatal(err)
	}
	renderer.tempDir = td
	renderer.workDir = td

	renderer.Close()

	if _, err := os.Stat(td); !os.IsNotExist(err) {
		t.Error("temp dir should be removed after Close()")
	}
}

func TestMarkdownSession_GraphvizRendersAsImage(t *testing.T) {
	renderer := newTestGraphvizRenderer(t)

	session := New(Options{
		GraphvizOptions: &GraphvizOptions{DotPath: renderer.dotPath},
	})
	// replace the auto-created graphviz renderer with our test one
	session.graphvizRenderer = renderer

	md := "# Title\n\n```dot\ndigraph { A -> B }\n```\n\nSome text.\n"
	if err := session.SetMarkdown(md); err != nil {
		t.Fatalf("SetMarkdown: %v", err)
	}

	var images int
	for _, elem := range session.Elements() {
		if elem.Type == NavElementImage {
			images++
			if elem.Text != "dot diagram" {
				t.Errorf("image text: got %q, want %q", elem.Text, "dot diagram")
			}
			if !strings.HasSuffix(elem.URL, ".png") {
				t.Errorf("image URL should end with .png: %q", elem.URL)
			}
		}
	}
	if images != 1 {
		t.Errorf("expected 1 image element, got %d", images)
	}

	joined := strings.Join(session.RenderedLines(), "\n")
	if !strings.Contains(joined, "[image: dot diagram]") {
		t.Errorf("expected fallback text in output, got:\n%s", joined)
	}
}

func TestMarkdownSession_MixedMermaidAndGraphviz(t *testing.T) {
	mermaidRenderer := newTestMermaidRenderer(t)
	graphvizRenderer := newTestGraphvizRenderer(t)

	session := New(Options{
		MermaidOptions:  &MermaidOptions{MmdcPath: mermaidRenderer.mmdcPath},
		GraphvizOptions: &GraphvizOptions{DotPath: graphvizRenderer.dotPath},
	})
	session.mermaidRenderer = mermaidRenderer
	session.graphvizRenderer = graphvizRenderer

	md := "# Mixed\n\n```mermaid\ngraph TD\n    A-->B\n```\n\n```dot\ndigraph { C -> D }\n```\n"
	if err := session.SetMarkdown(md); err != nil {
		t.Fatalf("SetMarkdown: %v", err)
	}

	var images int
	for _, elem := range session.Elements() {
		if elem.Type == NavElementImage {
			images++
		}
	}
	if images != 2 {
		t.Errorf("expected 2 image elements (mermaid + graphviz), got %d", images)
	}
}
