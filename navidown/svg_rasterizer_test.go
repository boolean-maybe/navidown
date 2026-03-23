package navidown

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestIsSVGData(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want bool
	}{
		{"svg tag", []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`), true},
		{"xml preamble with svg", []byte(`<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg"></svg>`), true},
		{"BOM prefix", append([]byte{0xEF, 0xBB, 0xBF}, []byte(`<svg></svg>`)...), true},
		{"whitespace prefix", []byte("  \n\t<svg></svg>"), true},
		{"png bytes", []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, false},
		{"plain text", []byte("hello world"), false},
		{"xml without svg", []byte(`<?xml version="1.0"?><html></html>`), false},
		{"empty", []byte{}, false},
		{"svg in middle without xml", []byte(`some text <svg></svg>`), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSVGData(tt.data)
			if got != tt.want {
				t.Errorf("isSVGData() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResvgRasterizer(t *testing.T) {
	if _, err := exec.LookPath("resvg"); err != nil {
		t.Skip("resvg not found in PATH, skipping integration test")
	}

	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100">
		<rect width="100" height="100" fill="red"/>
	</svg>`)

	rast := &ResvgRasterizer{}
	pngData, err := rast.Rasterize(svg, 200)
	if err != nil {
		t.Fatalf("Rasterize: %v", err)
	}

	// verify PNG header
	if !bytes.HasPrefix(pngData, []byte{0x89, 0x50, 0x4E, 0x47}) {
		t.Fatal("output does not have PNG header")
	}

	// verify decodable dimensions
	cfg, err := png.DecodeConfig(bytes.NewReader(pngData))
	if err != nil {
		t.Fatalf("png.DecodeConfig: %v", err)
	}
	if cfg.Width != 200 {
		t.Errorf("width = %d, want 200", cfg.Width)
	}
	if cfg.Height <= 0 {
		t.Errorf("height = %d, want > 0", cfg.Height)
	}
}

// make1x1PNG creates a minimal valid 1x1 red PNG for test use.
func make1x1PNG() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, image.Black)
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

type mockSVGRasterizer struct {
	called    bool
	callCount int
	width     int
	pngData   []byte
	err       error
}

func (m *mockSVGRasterizer) Rasterize(_ []byte, targetWidth int) ([]byte, error) {
	m.called = true
	m.callCount++
	m.width = targetWidth
	return m.pngData, m.err
}

func TestImageResolver_SVG(t *testing.T) {
	// write a temp SVG file
	dir := t.TempDir()
	svgPath := filepath.Join(dir, "test.svg")
	svgContent := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="100" height="50"></svg>`)
	if err := os.WriteFile(svgPath, svgContent, 0644); err != nil {
		t.Fatal(err)
	}

	pngBytes := make1x1PNG()
	mock := &mockSVGRasterizer{pngData: pngBytes}

	resolver := NewImageResolver([]string{dir})
	resolver.SetSVGRasterizer(mock)

	info, err := resolver.Resolve("test.svg", filepath.Join(dir, "doc.md"))
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if !mock.called {
		t.Fatal("expected mock rasterizer to be called")
	}
	// SVG has width="100", so rasterize at 100 * defaultSVGScaleFactor = 100
	expectedWidth := 100 * defaultSVGScaleFactor
	if mock.width != expectedWidth {
		t.Errorf("rasterizer width = %d, want %d", mock.width, expectedWidth)
	}
	if info.Width != 1 || info.Height != 1 {
		t.Errorf("dimensions = %dx%d, want 1x1", info.Width, info.Height)
	}
	if info.Format != "png" {
		t.Errorf("format = %q, want %q", info.Format, "png")
	}
}

func TestImageResolver_SVG_CustomWidth(t *testing.T) {
	dir := t.TempDir()
	svgPath := filepath.Join(dir, "test.svg")
	// SVG without explicit dimensions — falls back to svgRasterWidth
	svgContent := []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`)
	if err := os.WriteFile(svgPath, svgContent, 0644); err != nil {
		t.Fatal(err)
	}

	pngBytes := make1x1PNG()
	mock := &mockSVGRasterizer{pngData: pngBytes}

	resolver := NewImageResolver([]string{dir})
	resolver.SetSVGRasterizer(mock)
	resolver.SetSVGRasterWidth(1024)

	_, err := resolver.Resolve("test.svg", filepath.Join(dir, "doc.md"))
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if mock.width != 1024 {
		t.Errorf("rasterizer width = %d, want 1024", mock.width)
	}
}

func TestImageResolver_SVG_RasterizerError(t *testing.T) {
	dir := t.TempDir()
	svgPath := filepath.Join(dir, "bad.svg")
	svgContent := []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`)
	if err := os.WriteFile(svgPath, svgContent, 0644); err != nil {
		t.Fatal(err)
	}

	mock := &mockSVGRasterizer{
		err: os.ErrNotExist, // arbitrary error
	}

	resolver := NewImageResolver([]string{dir})
	resolver.SetSVGRasterizer(mock)

	_, err := resolver.Resolve("bad.svg", filepath.Join(dir, "doc.md"))
	if err == nil {
		t.Fatal("expected error from failing rasterizer")
	}
	if !mock.called {
		t.Fatal("expected mock rasterizer to be called")
	}
}

func TestParseSVGDimensions(t *testing.T) {
	tests := []struct {
		name          string
		svg           string
		wantW, wantH  float64
		wantOK        bool
		approxCompare bool // use approximate comparison for unit conversions
	}{
		{
			name:  "bare numbers",
			svg:   `<svg xmlns="http://www.w3.org/2000/svg" width="90" height="20"></svg>`,
			wantW: 90, wantH: 20, wantOK: true,
		},
		{
			name:  "px units",
			svg:   `<svg xmlns="http://www.w3.org/2000/svg" width="90px" height="20px"></svg>`,
			wantW: 90, wantH: 20, wantOK: true,
		},
		{
			name:  "percent with viewBox fallback",
			svg:   `<svg xmlns="http://www.w3.org/2000/svg" width="100%" height="100%" viewBox="0 0 90 20"></svg>`,
			wantW: 90, wantH: 20, wantOK: true,
		},
		{
			name:  "viewBox only",
			svg:   `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 200 100"></svg>`,
			wantW: 200, wantH: 100, wantOK: true,
		},
		{
			name:  "no dimensions",
			svg:   `<svg xmlns="http://www.w3.org/2000/svg"></svg>`,
			wantW: 0, wantH: 0, wantOK: false,
		},
		{
			name:  "pt units",
			svg:   `<svg xmlns="http://www.w3.org/2000/svg" width="72pt" height="36pt"></svg>`,
			wantW: 96, wantH: 48, wantOK: true,
			approxCompare: true,
		},
		{
			name:  "in units",
			svg:   `<svg xmlns="http://www.w3.org/2000/svg" width="1in" height="0.5in"></svg>`,
			wantW: 96, wantH: 48, wantOK: true,
			approxCompare: true,
		},
		{
			name:  "viewBox with offset",
			svg:   `<svg xmlns="http://www.w3.org/2000/svg" viewBox="10 20 300 150"></svg>`,
			wantW: 300, wantH: 150, wantOK: true,
		},
		{
			name:  "xml preamble",
			svg:   `<?xml version="1.0" encoding="UTF-8"?><svg xmlns="http://www.w3.org/2000/svg" width="50" height="25"></svg>`,
			wantW: 50, wantH: 25, wantOK: true,
		},
		{
			name:  "em units fall back to viewBox",
			svg:   `<svg xmlns="http://www.w3.org/2000/svg" width="10em" height="5em" viewBox="0 0 160 80"></svg>`,
			wantW: 160, wantH: 80, wantOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, h, ok := parseSVGDimensions([]byte(tt.svg))
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if tt.approxCompare {
				if diff := w - tt.wantW; diff < -0.5 || diff > 0.5 {
					t.Errorf("width = %f, want ~%f", w, tt.wantW)
				}
				if diff := h - tt.wantH; diff < -0.5 || diff > 0.5 {
					t.Errorf("height = %f, want ~%f", h, tt.wantH)
				}
			} else {
				if w != tt.wantW {
					t.Errorf("width = %f, want %f", w, tt.wantW)
				}
				if h != tt.wantH {
					t.Errorf("height = %f, want %f", h, tt.wantH)
				}
			}
		})
	}
}

func TestImageResolver_SVG_IntrinsicDimensions(t *testing.T) {
	dir := t.TempDir()

	// SVG with intrinsic width=90 → should rasterize at 90*2=180
	svgPath := filepath.Join(dir, "badge.svg")
	svgContent := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="90" height="20"></svg>`)
	if err := os.WriteFile(svgPath, svgContent, 0644); err != nil {
		t.Fatal(err)
	}

	pngBytes := make1x1PNG()
	mock := &mockSVGRasterizer{pngData: pngBytes}

	resolver := NewImageResolver([]string{dir})
	resolver.SetSVGRasterizer(mock)

	_, err := resolver.Resolve("badge.svg", filepath.Join(dir, "doc.md"))
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	expectedWidth := 90 * defaultSVGScaleFactor
	if mock.width != expectedWidth {
		t.Errorf("rasterizer width = %d, want %d", mock.width, expectedWidth)
	}
}

func TestImageResolver_SVG_NoDimensions_FallsBack(t *testing.T) {
	dir := t.TempDir()

	// SVG without dimensions → should fall back to defaultSVGRasterWidth
	svgPath := filepath.Join(dir, "nodim.svg")
	svgContent := []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`)
	if err := os.WriteFile(svgPath, svgContent, 0644); err != nil {
		t.Fatal(err)
	}

	pngBytes := make1x1PNG()
	mock := &mockSVGRasterizer{pngData: pngBytes}

	resolver := NewImageResolver([]string{dir})
	resolver.SetSVGRasterizer(mock)

	_, err := resolver.Resolve("nodim.svg", filepath.Join(dir, "doc.md"))
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if mock.width != defaultSVGRasterWidth {
		t.Errorf("rasterizer width = %d, want %d (fallback)", mock.width, defaultSVGRasterWidth)
	}
}

func TestImageResolver_SVG_CustomScaleFactor(t *testing.T) {
	dir := t.TempDir()

	svgPath := filepath.Join(dir, "badge.svg")
	svgContent := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="90" height="20"></svg>`)
	if err := os.WriteFile(svgPath, svgContent, 0644); err != nil {
		t.Fatal(err)
	}

	pngBytes := make1x1PNG()
	mock := &mockSVGRasterizer{pngData: pngBytes}

	resolver := NewImageResolver([]string{dir})
	resolver.SetSVGRasterizer(mock)
	resolver.SetSVGScaleFactor(3)

	_, err := resolver.Resolve("badge.svg", filepath.Join(dir, "doc.md"))
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if mock.width != 270 {
		t.Errorf("rasterizer width = %d, want 270 (90 * 3)", mock.width)
	}
}

func TestCachingSVGRasterizer_DiskCache(t *testing.T) {
	pngBytes := make1x1PNG()
	mock := &mockSVGRasterizer{pngData: pngBytes}
	cacheDir := t.TempDir()

	c := NewCachingSVGRasterizer(mock, cacheDir)
	if c == nil {
		t.Fatal("NewCachingSVGRasterizer returned nil")
	}

	svgData := []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`)

	// first call: should invoke inner rasterizer
	result1, err := c.Rasterize(svgData, 200)
	if err != nil {
		t.Fatalf("first Rasterize: %v", err)
	}
	if !mock.called {
		t.Fatal("expected inner rasterizer to be called")
	}
	if mock.callCount != 1 {
		t.Errorf("call count = %d, want 1", mock.callCount)
	}

	// verify disk file was written
	key := svgCacheKey(svgData, 200)
	diskPath := filepath.Join(cacheDir, key+".png")
	if _, err := os.Stat(diskPath); err != nil {
		t.Fatalf("disk cache file not found: %v", err)
	}

	// second call: should hit in-memory cache, no additional inner call
	result2, err := c.Rasterize(svgData, 200)
	if err != nil {
		t.Fatalf("second Rasterize: %v", err)
	}
	if mock.callCount != 1 {
		t.Errorf("call count = %d after in-memory cache hit, want 1", mock.callCount)
	}
	if !bytes.Equal(result1, result2) {
		t.Error("results differ between first and cached call")
	}

	// new instance with same cache dir: should hit disk cache
	mock2 := &mockSVGRasterizer{pngData: pngBytes}
	c2 := NewCachingSVGRasterizer(mock2, cacheDir)
	if c2 == nil {
		t.Fatal("c2 is nil")
	}

	result3, err := c2.Rasterize(svgData, 200)
	if err != nil {
		t.Fatalf("disk cache Rasterize: %v", err)
	}
	if mock2.callCount != 0 {
		t.Errorf("inner rasterizer called %d times on disk cache hit, want 0", mock2.callCount)
	}
	if !bytes.Equal(result1, result3) {
		t.Error("disk-cached result differs from original")
	}
}

func TestCachingSVGRasterizer_Close(t *testing.T) {
	pngBytes := make1x1PNG()
	mock := &mockSVGRasterizer{pngData: pngBytes}

	// force temp dir by creating the caching rasterizer with workDir set to a temp dir
	c := &CachingSVGRasterizer{inner: mock}
	td, err := os.MkdirTemp("", "navidown-svg-test-")
	if err != nil {
		t.Fatal(err)
	}
	c.tempDir = td
	c.workDir = td

	c.Close()

	if _, err := os.Stat(td); !os.IsNotExist(err) {
		t.Error("temp dir should be removed after Close()")
	}
}

func TestCachingSVGRasterizer_DifferentWidths(t *testing.T) {
	pngBytes := make1x1PNG()
	mock := &mockSVGRasterizer{pngData: pngBytes}
	cacheDir := t.TempDir()

	c := NewCachingSVGRasterizer(mock, cacheDir)
	if c == nil {
		t.Fatal("nil")
	}

	svgData := []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`)

	if _, err := c.Rasterize(svgData, 200); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Rasterize(svgData, 400); err != nil {
		t.Fatal(err)
	}

	// should have called inner twice (different cache keys)
	if mock.callCount != 2 {
		t.Errorf("call count = %d, want 2 (different widths)", mock.callCount)
	}
}

func TestImageResolver_PreResolve(t *testing.T) {
	dir := t.TempDir()
	pngBytes := make1x1PNG()

	// write multiple image files
	for _, name := range []string{"a.png", "b.png", "c.png"} {
		if err := os.WriteFile(filepath.Join(dir, name), pngBytes, 0644); err != nil {
			t.Fatal(err)
		}
	}

	resolver := NewImageResolver([]string{dir})
	sourceFile := filepath.Join(dir, "doc.md")

	// pre-resolve all
	resolver.PreResolve([]string{"a.png", "b.png", "c.png"}, sourceFile)

	// verify all are cached (Resolve should return immediately with no error)
	for _, name := range []string{"a.png", "b.png", "c.png"} {
		info, err := resolver.Resolve(name, sourceFile)
		if err != nil {
			t.Errorf("Resolve(%q) after PreResolve: %v", name, err)
			continue
		}
		if info == nil {
			t.Errorf("Resolve(%q) returned nil info", name)
		}
	}
}

func TestImageResolver_PNGNotRasterized(t *testing.T) {
	// ensure non-SVG data does NOT trigger the rasterizer
	dir := t.TempDir()
	pngBytes := make1x1PNG()
	pngPath := filepath.Join(dir, "test.png")
	if err := os.WriteFile(pngPath, pngBytes, 0644); err != nil {
		t.Fatal(err)
	}

	mock := &mockSVGRasterizer{pngData: pngBytes}

	resolver := NewImageResolver([]string{dir})
	resolver.SetSVGRasterizer(mock)

	info, err := resolver.Resolve("test.png", filepath.Join(dir, "doc.md"))
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if mock.called {
		t.Fatal("rasterizer should NOT be called for PNG data")
	}
	if info.Format != "png" {
		t.Errorf("format = %q, want %q", info.Format, "png")
	}
}
