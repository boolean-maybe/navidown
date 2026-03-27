package navidown

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// newTestMermaidRenderer creates a MermaidRenderer that uses a fake mmdc script
// which copies a 1x1 PNG fixture to the output path.
func newTestMermaidRenderer(t *testing.T) *MermaidRenderer {
	t.Helper()

	fixturePNG := minimalPNG()
	scriptDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(scriptDir, "fixture.png"), fixturePNG, 0644); err != nil {
		t.Fatalf("write fixture PNG: %v", err)
	}

	scriptPath := writeFakeMmdc(t, scriptDir, filepath.Join(scriptDir, "fixture.png"))

	cacheDir := t.TempDir()
	renderer := NewMermaidRenderer(MermaidOptions{MmdcPath: scriptPath, CacheDir: cacheDir})
	if renderer == nil {
		t.Fatal("NewMermaidRenderer returned nil")
	}
	t.Cleanup(renderer.Close)
	return renderer
}

// writeFakeMmdc creates a platform-appropriate fake mmdc executable that copies
// fixturePath to the -o argument. Returns the path to the executable.
func writeFakeMmdc(t *testing.T, dir, fixturePath string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		return writeFakeMmdcBat(t, dir, fixturePath)
	}
	return writeFakeMmdcSh(t, dir, fixturePath)
}

func writeFakeMmdcSh(t *testing.T, dir, fixturePath string) string {
	t.Helper()
	scriptPath := filepath.Join(dir, "fake-mmdc")
	script := fmt.Sprintf(`#!/bin/sh
while [ $# -gt 0 ]; do
  case "$1" in
    -o) shift; cp "%s" "$1" ;;
  esac
  shift
done
`, fixturePath)
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("write fake mmdc: %v", err)
	}
	return scriptPath
}

func writeFakeMmdcBat(t *testing.T, dir, fixturePath string) string {
	t.Helper()
	batPath := filepath.Join(dir, "fake-mmdc.bat")
	fixturePath = filepath.FromSlash(fixturePath)
	script := fmt.Sprintf("@echo off\r\n:loop\r\nif \"%%~1\"==\"\" goto end\r\nif \"%%~1\"==\"-o\" (\r\n  copy /Y \"%s\" \"%%~2\" >nul\r\n  shift\r\n)\r\nshift\r\ngoto loop\r\n:end\r\n", fixturePath)
	if err := os.WriteFile(batPath, []byte(script), 0755); err != nil {
		t.Fatalf("write fake mmdc.bat: %v", err)
	}
	return batPath
}

// dummyExecutable returns a path to an executable that exists on all platforms.
// Used when tests need a valid MmdcPath but never actually invoke it.
func dummyExecutable() string {
	if runtime.GOOS == "windows" {
		return "cmd.exe"
	}
	return "/bin/echo"
}

// writeFailingMmdc creates a platform-appropriate fake mmdc that always exits with code 1.
func writeFailingMmdc(t *testing.T, dir string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		p := filepath.Join(dir, "bad-mmdc.bat")
		if err := os.WriteFile(p, []byte("@echo off\r\nexit /b 1\r\n"), 0755); err != nil {
			t.Fatalf("write bad mmdc.bat: %v", err)
		}
		return p
	}
	p := filepath.Join(dir, "bad-mmdc")
	if err := os.WriteFile(p, []byte("#!/bin/sh\nexit 1\n"), 0755); err != nil {
		t.Fatalf("write bad mmdc: %v", err)
	}
	return p
}

// writeFakeMmdcWithCounter creates a fake mmdc that copies fixturePath to the -o
// argument AND increments a counter file on each invocation.
// Returns (scriptPath, counterPath).
func writeFakeMmdcWithCounter(t *testing.T, dir, fixturePath string) (string, string) {
	t.Helper()
	counterPath := filepath.Join(dir, "counter")
	if runtime.GOOS == "windows" {
		return writeFakeMmdcWithCounterBat(t, dir, fixturePath, counterPath), counterPath
	}
	return writeFakeMmdcWithCounterSh(t, dir, fixturePath, counterPath), counterPath
}

func writeFakeMmdcWithCounterSh(t *testing.T, dir, fixturePath, counterPath string) string {
	t.Helper()
	scriptPath := filepath.Join(dir, "fake-mmdc")
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
		t.Fatalf("write fake mmdc: %v", err)
	}
	return scriptPath
}

func writeFakeMmdcWithCounterBat(t *testing.T, dir, fixturePath, counterPath string) string {
	t.Helper()
	batPath := filepath.Join(dir, "fake-mmdc.bat")
	fixturePath = filepath.FromSlash(fixturePath)
	counterPath = filepath.FromSlash(counterPath)
	// batch script that increments a counter file and copies fixture to -o target
	script := fmt.Sprintf("@echo off\r\nsetlocal enabledelayedexpansion\r\nset count=0\r\nif exist \"%s\" (\r\n  set /p count=<\"%s\"\r\n)\r\nset /a count=count+1\r\necho !count!>\"%s\"\r\n:loop\r\nif \"%%~1\"==\"\" goto end\r\nif \"%%~1\"==\"-o\" (\r\n  copy /Y \"%s\" \"%%~2\" >nul\r\n  shift\r\n)\r\nshift\r\ngoto loop\r\n:end\r\n", counterPath, counterPath, counterPath, fixturePath)
	if err := os.WriteFile(batPath, []byte(script), 0755); err != nil {
		t.Fatalf("write fake mmdc.bat: %v", err)
	}
	return batPath
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
	scriptDir := t.TempDir()
	scriptPath := writeFailingMmdc(t, scriptDir)

	cacheDir := t.TempDir()
	renderer := NewMermaidRenderer(MermaidOptions{MmdcPath: scriptPath, CacheDir: cacheDir})
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

func TestMermaidRenderer_Close_PersistentDir(t *testing.T) {
	persistentDir := t.TempDir()

	// use any executable that exists on all platforms as a dummy mmdcPath
	dummyExe := dummyExecutable()

	renderer := NewMermaidRenderer(MermaidOptions{
		MmdcPath: dummyExe,
		CacheDir: persistentDir,
	})
	if renderer == nil {
		t.Fatal("NewMermaidRenderer returned nil")
	}

	renderer.Close()

	// persistent dir should NOT be removed
	if _, err := os.Stat(persistentDir); err != nil {
		t.Error("persistent cache dir should survive Close()")
	}
}

func TestMermaidRenderer_Close_TempDir(t *testing.T) {
	// force temp dir by providing an invalid explicit path and mocking UserCacheDir
	// We can't easily mock UserCacheDir, so we test the tempDir path directly.
	renderer := &MermaidRenderer{
		opts:     MermaidOptions{},
		mmdcPath: dummyExecutable(),
	}

	td, err := os.MkdirTemp("", "navidown-mermaid-test-")
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

func TestExtractMermaidBlocks(t *testing.T) {
	tests := []struct {
		name       string
		markdown   string
		wantCount  int
		wantSource []string // expected source for each block
	}{
		{
			name:       "single block",
			markdown:   "```mermaid\ngraph TD\n    A-->B\n```\n",
			wantCount:  1,
			wantSource: []string{"graph TD\n    A-->B\n"},
		},
		{
			name:       "multiple blocks",
			markdown:   "```mermaid\ngraph TD\n```\n\ntext\n\n```mermaid\nsequenceDiagram\n```\n",
			wantCount:  2,
			wantSource: []string{"graph TD\n", "sequenceDiagram\n"},
		},
		{
			name:      "unclosed fence",
			markdown:  "```mermaid\ngraph TD\n    A-->B\n",
			wantCount: 0,
		},
		{
			name:       "mixed with non-mermaid",
			markdown:   "```go\nfunc main(){}\n```\n\n```mermaid\ngraph LR\n```\n",
			wantCount:  1,
			wantSource: []string{"graph LR\n"},
		},
		{
			name:      "no mermaid blocks",
			markdown:  "# Title\n\nSome text.\n",
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, blocks := extractMermaidBlocks(tt.markdown)
			if len(blocks) != tt.wantCount {
				t.Fatalf("got %d blocks, want %d", len(blocks), tt.wantCount)
			}
			for i, want := range tt.wantSource {
				if blocks[i].source != want {
					t.Errorf("block %d: got source %q, want %q", i, blocks[i].source, want)
				}
			}
		})
	}
}

func TestReassembleMermaid(t *testing.T) {
	t.Run("substitutes rendered blocks", func(t *testing.T) {
		md := "# Title\n\n```mermaid\ngraph TD\n```\n\nEnd.\n"
		lines, blocks := extractMermaidBlocks(md)
		rendered := map[int]string{0: "/tmp/test.png"}

		result := reassembleMermaid(lines, blocks, rendered)
		if !strings.Contains(result, "![mermaid diagram](/tmp/test.png)") {
			t.Errorf("expected image substitution, got:\n%s", result)
		}
		if !strings.Contains(result, "# Title") {
			t.Error("title should be preserved")
		}
		if !strings.Contains(result, "End.") {
			t.Error("trailing text should be preserved")
		}
	})

	t.Run("preserves block on error", func(t *testing.T) {
		md := "```mermaid\ngraph TD\n```\n"
		lines, blocks := extractMermaidBlocks(md)
		rendered := map[int]string{} // empty = all errors

		result := reassembleMermaid(lines, blocks, rendered)
		if !strings.Contains(result, "```mermaid") {
			t.Error("original block should be preserved on error")
		}
	})

	t.Run("mixed success and error", func(t *testing.T) {
		md := "```mermaid\ngraph A\n```\n\n```mermaid\ngraph B\n```\n"
		lines, blocks := extractMermaidBlocks(md)
		rendered := map[int]string{0: "/tmp/a.png"} // second block failed

		result := reassembleMermaid(lines, blocks, rendered)
		if !strings.Contains(result, "![mermaid diagram](/tmp/a.png)") {
			t.Error("first block should be rendered")
		}
		if !strings.Contains(result, "```mermaid") {
			t.Error("second block should be preserved (error)")
		}
	})
}

func TestMermaidRenderer_DiskCache(t *testing.T) {
	fixturePNG := minimalPNG()
	scriptDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(scriptDir, "fixture.png"), fixturePNG, 0644); err != nil {
		t.Fatal(err)
	}
	scriptPath, counterPath := writeFakeMmdcWithCounter(t, scriptDir, filepath.Join(scriptDir, "fixture.png"))

	cacheDir := t.TempDir()
	source := "graph TD\n    A-->B\n"

	// first renderer: renders and writes to disk cache
	r1 := NewMermaidRenderer(MermaidOptions{MmdcPath: scriptPath, CacheDir: cacheDir})
	if r1 == nil {
		t.Fatal("r1 is nil")
	}
	path1, err := r1.RenderToFile(source)
	if err != nil {
		t.Fatalf("r1 render: %v", err)
	}
	r1.Close()

	// second renderer: should find PNG on disk, no mmdc call
	r2 := NewMermaidRenderer(MermaidOptions{MmdcPath: scriptPath, CacheDir: cacheDir})
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

	// counter should be 1 — mmdc was only called once
	data, err := os.ReadFile(counterPath)
	if err != nil {
		t.Fatalf("read counter: %v", err)
	}
	if strings.TrimSpace(string(data)) != "1" {
		t.Errorf("mmdc was called %s times, expected 1", strings.TrimSpace(string(data)))
	}
}

func TestMermaidRenderer_CacheKeyIncludesOptions(t *testing.T) {
	opts1 := MermaidOptions{Theme: "dark"}
	opts2 := MermaidOptions{Theme: "forest"}
	r1 := &MermaidRenderer{opts: opts1, configData: opts1.resolvedConfigData()}
	r2 := &MermaidRenderer{opts: opts2, configData: opts2.resolvedConfigData()}

	source := "graph TD\n    A-->B\n"
	k1 := r1.cacheKey(source)
	k2 := r2.cacheKey(source)

	if k1 == k2 {
		t.Error("different themes should produce different cache keys")
	}

	// same options = same key
	opts3 := MermaidOptions{Theme: "dark"}
	r3 := &MermaidRenderer{opts: opts3, configData: opts3.resolvedConfigData()}
	k3 := r3.cacheKey(source)
	if k1 != k3 {
		t.Error("same options should produce same cache key")
	}
}

func TestResolvedConfigData(t *testing.T) {
	t.Run("valid JSON with correct theme", func(t *testing.T) {
		opts := MermaidOptions{Theme: "forest"}
		data := opts.resolvedConfigData()

		var cfg map[string]any
		if err := json.Unmarshal(data, &cfg); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if cfg["theme"] != "forest" {
			t.Errorf("theme: got %q, want %q", cfg["theme"], "forest")
		}
	})

	t.Run("default theme is base", func(t *testing.T) {
		opts := MermaidOptions{}
		data := opts.resolvedConfigData()

		var cfg map[string]any
		if err := json.Unmarshal(data, &cfg); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if cfg["theme"] != "base" {
			t.Errorf("theme: got %q, want %q", cfg["theme"], "base")
		}
	})

	t.Run("different themes produce different bytes", func(t *testing.T) {
		o1 := MermaidOptions{Theme: "dark"}
		o2 := MermaidOptions{Theme: "forest"}
		d1 := o1.resolvedConfigData()
		d2 := o2.resolvedConfigData()
		if string(d1) == string(d2) {
			t.Error("different themes should produce different config data")
		}
	})

	t.Run("deterministic output", func(t *testing.T) {
		opts := MermaidOptions{Theme: "dark"}
		d1 := opts.resolvedConfigData()
		d2 := opts.resolvedConfigData()
		if string(d1) != string(d2) {
			t.Error("same options should produce identical config data")
		}
	})
}

func TestResolveCacheDir(t *testing.T) {
	t.Run("explicit path", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "explicit")
		persistent, temp, work := resolveCacheDir(dir, "mermaid")
		if persistent != dir {
			t.Errorf("persistentDir: got %q, want %q", persistent, dir)
		}
		if temp != "" {
			t.Errorf("tempDir should be empty for explicit path, got %q", temp)
		}
		if work != dir {
			t.Errorf("workDir: got %q, want %q", work, dir)
		}
		// dir should exist
		if _, err := os.Stat(dir); err != nil {
			t.Errorf("explicit dir should have been created: %v", err)
		}
	})

	t.Run("auto path (empty string)", func(t *testing.T) {
		persistent, temp, work := resolveCacheDir("", "mermaid")
		// should succeed with either persistent or temp
		if work == "" {
			t.Fatal("resolveCacheDir should find a usable dir")
		}
		if persistent != "" {
			// persistent path should contain navidown/mermaid
			if !strings.Contains(persistent, filepath.Join("navidown", "mermaid")) {
				t.Errorf("persistent dir should contain navidown/mermaid: %q", persistent)
			}
			if temp != "" {
				t.Error("temp should be empty when persistent succeeds")
			}
		} else {
			// fell back to temp
			if temp == "" {
				t.Error("if no persistent, temp should be set")
			}
		}
	})
}
