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
	called  bool
	width   int
	pngData []byte
	err     error
}

func (m *mockSVGRasterizer) Rasterize(_ []byte, targetWidth int) ([]byte, error) {
	m.called = true
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
	if mock.width != defaultSVGRasterWidth {
		t.Errorf("rasterizer width = %d, want %d", mock.width, defaultSVGRasterWidth)
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
	svgContent := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="100" height="50"></svg>`)
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
