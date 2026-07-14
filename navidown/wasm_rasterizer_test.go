package navidown

import (
	"bytes"
	"image/png"
	"testing"
)

func TestWasmRasterizerScalesToTargetWidth(t *testing.T) {
	// intrinsic 60x30 -> target width 120 -> expect 120x60 png
	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="60" height="30"><rect width="60" height="30" fill="red"/></svg>`)
	rast, err := sharedWasmRasterizer()
	if err != nil {
		t.Fatal(err)
	}
	out, err := rast.Rasterize(svg, 120)
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("not a valid png: %v", err)
	}
	if img.Bounds().Dx() != 120 || img.Bounds().Dy() != 60 {
		t.Fatalf("bounds %v, want 120x60", img.Bounds())
	}
}

func TestWasmRasterizerInvalidSVG(t *testing.T) {
	rast, _ := sharedWasmRasterizer()
	_, err := rast.Rasterize([]byte("not an svg"), 100)
	if err == nil {
		t.Fatal("want error for invalid svg")
	}
}
