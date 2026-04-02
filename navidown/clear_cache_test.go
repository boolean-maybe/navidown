package navidown

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCachingSVGRasterizer_ClearCache(t *testing.T) {
	pngBytes := make1x1PNG()
	mock := &mockSVGRasterizer{pngData: pngBytes}
	cacheDir := t.TempDir()

	c := NewCachingSVGRasterizer(mock, cacheDir)
	if c == nil {
		t.Fatal("NewCachingSVGRasterizer returned nil")
	}

	svgData := []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`)

	// populate cache
	if _, err := c.Rasterize(svgData, 200); err != nil {
		t.Fatalf("Rasterize: %v", err)
	}
	if mock.callCount != 1 {
		t.Fatalf("expected 1 call, got %d", mock.callCount)
	}

	// verify disk PNG exists
	key := svgCacheKey(svgData, 200)
	diskPath := filepath.Join(cacheDir, key+".png")
	if _, err := os.Stat(diskPath); err != nil {
		t.Fatalf("disk cache file should exist: %v", err)
	}

	c.ClearCache()

	// disk PNG should be gone
	if _, err := os.Stat(diskPath); !os.IsNotExist(err) {
		t.Error("disk cache file should be removed after ClearCache()")
	}

	// next call should miss cache and call inner rasterizer again
	if _, err := c.Rasterize(svgData, 200); err != nil {
		t.Fatalf("Rasterize after clear: %v", err)
	}
	if mock.callCount != 2 {
		t.Errorf("expected 2 calls after cache clear, got %d", mock.callCount)
	}
}

func TestImageResolver_ClearCache(t *testing.T) {
	dir := t.TempDir()
	pngBytes := make1x1PNG()

	imgPath := filepath.Join(dir, "test.png")
	if err := os.WriteFile(imgPath, pngBytes, 0644); err != nil {
		t.Fatal(err)
	}

	resolver := NewImageResolver([]string{dir})
	sourceFile := filepath.Join(dir, "doc.md")

	// populate cache
	info1, err := resolver.Resolve("test.png", sourceFile)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	resolver.ClearCache()

	// write a different image to the same path
	newPNG := make1x1PNG() // same bytes but resolver should re-read from disk
	if err := os.WriteFile(imgPath, newPNG, 0644); err != nil {
		t.Fatal(err)
	}

	info2, err := resolver.Resolve("test.png", sourceFile)
	if err != nil {
		t.Fatalf("Resolve after clear: %v", err)
	}

	// both should succeed — the point is that Resolve works after ClearCache
	if info1 == nil || info2 == nil {
		t.Fatal("expected non-nil info from both resolves")
	}
}

func TestImageResolver_ClearCache_DelegatesSVG(t *testing.T) {
	dir := t.TempDir()
	pngBytes := make1x1PNG()
	mock := &mockSVGRasterizer{pngData: pngBytes}
	cacheDir := t.TempDir()

	cachingRast := NewCachingSVGRasterizer(mock, cacheDir)
	if cachingRast == nil {
		t.Fatal("NewCachingSVGRasterizer returned nil")
	}

	resolver := NewImageResolver([]string{dir})
	resolver.SetSVGRasterizer(cachingRast)

	// write an SVG file
	svgPath := filepath.Join(dir, "icon.svg")
	svgContent := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="10" height="10"></svg>`)
	if err := os.WriteFile(svgPath, svgContent, 0644); err != nil {
		t.Fatal(err)
	}

	// resolve to populate both caches
	if _, err := resolver.Resolve("icon.svg", filepath.Join(dir, "doc.md")); err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if mock.callCount != 1 {
		t.Fatalf("expected 1 rasterizer call, got %d", mock.callCount)
	}

	resolver.ClearCache()

	// re-resolve: should miss both caches and call rasterizer again
	if _, err := resolver.Resolve("icon.svg", filepath.Join(dir, "doc.md")); err != nil {
		t.Fatalf("Resolve after clear: %v", err)
	}
	if mock.callCount != 2 {
		t.Errorf("expected 2 rasterizer calls after clear, got %d", mock.callCount)
	}
}

func TestMermaidRenderer_ClearCache(t *testing.T) {
	renderer := newTestMermaidRenderer(t)

	source := "graph TD\n    A-->B\n"
	path1, err := renderer.RenderToFile(source)
	if err != nil {
		t.Fatalf("RenderToFile: %v", err)
	}

	// verify PNG on disk
	if _, err := os.Stat(path1); err != nil {
		t.Fatalf("rendered PNG should exist: %v", err)
	}

	renderer.ClearCache()

	// disk PNG should be gone
	if _, err := os.Stat(path1); !os.IsNotExist(err) {
		t.Error("disk cache PNG should be removed after ClearCache()")
	}

	// re-render should succeed (re-invokes mmdc)
	path2, err := renderer.RenderToFile(source)
	if err != nil {
		t.Fatalf("RenderToFile after clear: %v", err)
	}
	if _, err := os.Stat(path2); err != nil {
		t.Errorf("re-rendered PNG should exist: %v", err)
	}
}

func TestGraphvizRenderer_ClearCache(t *testing.T) {
	renderer := newTestGraphvizRenderer(t)

	source := "digraph { A -> B }\n"
	path1, err := renderer.RenderToFile(source)
	if err != nil {
		t.Fatalf("RenderToFile: %v", err)
	}

	if _, err := os.Stat(path1); err != nil {
		t.Fatalf("rendered PNG should exist: %v", err)
	}

	renderer.ClearCache()

	if _, err := os.Stat(path1); !os.IsNotExist(err) {
		t.Error("disk cache PNG should be removed after ClearCache()")
	}

	path2, err := renderer.RenderToFile(source)
	if err != nil {
		t.Fatalf("RenderToFile after clear: %v", err)
	}
	if _, err := os.Stat(path2); err != nil {
		t.Errorf("re-rendered PNG should exist: %v", err)
	}
}

func TestMermaidRenderer_EvictKeys(t *testing.T) {
	renderer := newTestMermaidRenderer(t)

	sourceA := "graph TD\n    A-->B\n"
	sourceB := "graph TD\n    X-->Y\n"

	pathA, err := renderer.RenderToFile(sourceA)
	if err != nil {
		t.Fatalf("render A: %v", err)
	}
	pathB, err := renderer.RenderToFile(sourceB)
	if err != nil {
		t.Fatalf("render B: %v", err)
	}

	// evict only A's key
	keyA := strings.TrimSuffix(filepath.Base(pathA), ".png")
	renderer.EvictKeys([]string{keyA})

	if _, err := os.Stat(pathA); !os.IsNotExist(err) {
		t.Error("evicted PNG A should be removed")
	}
	if _, err := os.Stat(pathB); err != nil {
		t.Error("PNG B should still exist")
	}

	// re-render A should succeed (cache miss → re-invokes mmdc)
	pathA2, err := renderer.RenderToFile(sourceA)
	if err != nil {
		t.Fatalf("re-render A: %v", err)
	}
	if _, err := os.Stat(pathA2); err != nil {
		t.Error("re-rendered PNG A should exist")
	}
}

func TestGraphvizRenderer_EvictKeys(t *testing.T) {
	renderer := newTestGraphvizRenderer(t)

	sourceA := "digraph { A -> B }\n"
	sourceB := "digraph { X -> Y }\n"

	pathA, err := renderer.RenderToFile(sourceA)
	if err != nil {
		t.Fatalf("render A: %v", err)
	}
	pathB, err := renderer.RenderToFile(sourceB)
	if err != nil {
		t.Fatalf("render B: %v", err)
	}

	keyA := strings.TrimSuffix(filepath.Base(pathA), ".png")
	renderer.EvictKeys([]string{keyA})

	if _, err := os.Stat(pathA); !os.IsNotExist(err) {
		t.Error("evicted PNG A should be removed")
	}
	if _, err := os.Stat(pathB); err != nil {
		t.Error("PNG B should still exist")
	}

	pathA2, err := renderer.RenderToFile(sourceA)
	if err != nil {
		t.Fatalf("re-render A: %v", err)
	}
	if _, err := os.Stat(pathA2); err != nil {
		t.Error("re-rendered PNG A should exist")
	}
}

func TestImageResolver_ClearCacheForURLs(t *testing.T) {
	dir := t.TempDir()
	pngBytes := make1x1PNG()

	imgA := filepath.Join(dir, "a.png")
	imgB := filepath.Join(dir, "b.png")
	if err := os.WriteFile(imgA, pngBytes, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(imgB, pngBytes, 0644); err != nil {
		t.Fatal(err)
	}

	resolver := NewImageResolver([]string{dir})
	sourceFile := filepath.Join(dir, "doc.md")

	// populate both
	infoA1, err := resolver.Resolve("a.png", sourceFile)
	if err != nil {
		t.Fatalf("Resolve a: %v", err)
	}
	infoB1, err := resolver.Resolve("b.png", sourceFile)
	if err != nil {
		t.Fatalf("Resolve b: %v", err)
	}

	// evict only a.png
	resolver.ClearCacheForURLs([]string{"a.png"})

	// b.png should return same pointer (cache hit)
	infoB2, err := resolver.Resolve("b.png", sourceFile)
	if err != nil {
		t.Fatalf("Resolve b after clear: %v", err)
	}
	if infoB1 != infoB2 {
		t.Error("b.png should still be cached (same pointer)")
	}

	// a.png should be re-resolved (cache miss, new pointer)
	infoA2, err := resolver.Resolve("a.png", sourceFile)
	if err != nil {
		t.Fatalf("Resolve a after clear: %v", err)
	}
	if infoA1 == infoA2 {
		t.Error("a.png should have been re-resolved (different pointer)")
	}
}

func TestDiagramKeysForRenderer_StrictValidation(t *testing.T) {
	workDir := "/cache/navidown/mermaid"

	// valid 64-char hex hash
	validKey := "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	elements := []NavElement{
		{Type: NavElementImage, URL: workDir + "/" + validKey + ".png"},
		// wrong type — should be skipped
		{Type: NavElementURL, URL: workDir + "/" + validKey + ".png"},
		// not in workDir — should be skipped
		{Type: NavElementImage, URL: "/other/dir/" + validKey + ".png"},
		// non-hex basename — should be skipped
		{Type: NavElementImage, URL: workDir + "/photo.png"},
		// short hash — should be skipped
		{Type: NavElementImage, URL: workDir + "/abcdef.png"},
		// prefix attack: workDir is a prefix of path but not a parent
		{Type: NavElementImage, URL: "/cache/navidown/mermaid-old/" + validKey + ".png"},
	}

	keys := diagramKeysForRenderer(elements, workDir)
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d: %v", len(keys), keys)
	}
	if keys[0] != validKey {
		t.Errorf("expected key %q, got %q", validKey, keys[0])
	}
}

func TestDiagramKeysForRenderer_EmptyWorkDir(t *testing.T) {
	elements := []NavElement{
		{Type: NavElementImage, URL: "/some/path.png"},
	}
	keys := diagramKeysForRenderer(elements, "")
	if keys != nil {
		t.Errorf("expected nil for empty workDir, got %v", keys)
	}
}

func TestMarkdownSession_ClearCachesForDocument_SelectiveRefresh(t *testing.T) {
	mermaid := newTestMermaidRenderer(t)
	graphviz := newTestGraphvizRenderer(t)

	// render diagrams for "document A"
	mermaidA := "graph TD\n    A-->B\n"
	graphvizA := "digraph { A -> B }\n"
	pathMA, err := mermaid.RenderToFile(mermaidA)
	if err != nil {
		t.Fatalf("mermaid A: %v", err)
	}
	pathGA, err := graphviz.RenderToFile(graphvizA)
	if err != nil {
		t.Fatalf("graphviz A: %v", err)
	}

	// render diagrams for "document B" (different source)
	mermaidB := "graph TD\n    X-->Y\n"
	graphvizB := "digraph { X -> Y }\n"
	pathMB, err := mermaid.RenderToFile(mermaidB)
	if err != nil {
		t.Fatalf("mermaid B: %v", err)
	}
	pathGB, err := graphviz.RenderToFile(graphvizB)
	if err != nil {
		t.Fatalf("graphviz B: %v", err)
	}

	// set up session with document A's elements
	session := New(Options{})
	session.mermaidRenderer = mermaid
	session.graphvizRenderer = graphviz
	session.elements = []NavElement{
		{Type: NavElementImage, URL: pathMA},
		{Type: NavElementImage, URL: pathGA},
	}

	// selective clear for document A
	session.ClearCachesForDocument()

	// A's PNGs should be gone
	if _, err := os.Stat(pathMA); !os.IsNotExist(err) {
		t.Error("mermaid A cache should be cleared")
	}
	if _, err := os.Stat(pathGA); !os.IsNotExist(err) {
		t.Error("graphviz A cache should be cleared")
	}

	// B's PNGs should still exist (warm)
	if _, err := os.Stat(pathMB); err != nil {
		t.Error("mermaid B cache should still exist")
	}
	if _, err := os.Stat(pathGB); err != nil {
		t.Error("graphviz B cache should still exist")
	}

	// re-render A should succeed
	pathMA2, err := mermaid.RenderToFile(mermaidA)
	if err != nil {
		t.Fatalf("re-render mermaid A: %v", err)
	}
	if _, err := os.Stat(pathMA2); err != nil {
		t.Error("re-rendered mermaid A should exist")
	}
}

func TestMarkdownSession_ClearCachesForDocument_NilRenderers(t *testing.T) {
	session := New(Options{})
	// should not panic with nil renderers
	session.ClearCachesForDocument()
}

func TestMarkdownSession_ClearCaches(t *testing.T) {
	session := New(Options{})

	// nil renderers — should not panic
	session.ClearCaches()

	// with renderers
	mermaid := newTestMermaidRenderer(t)
	graphviz := newTestGraphvizRenderer(t)
	session.mermaidRenderer = mermaid
	session.graphvizRenderer = graphviz

	// populate caches
	mermaidPath, err := mermaid.RenderToFile("graph TD\n    A-->B\n")
	if err != nil {
		t.Fatalf("mermaid render: %v", err)
	}
	graphvizPath, err := graphviz.RenderToFile("digraph { A -> B }\n")
	if err != nil {
		t.Fatalf("graphviz render: %v", err)
	}

	session.ClearCaches()

	if _, err := os.Stat(mermaidPath); !os.IsNotExist(err) {
		t.Error("mermaid cache PNG should be cleared")
	}
	if _, err := os.Stat(graphvizPath); !os.IsNotExist(err) {
		t.Error("graphviz cache PNG should be cleared")
	}
}
