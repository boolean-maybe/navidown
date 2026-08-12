package navidown

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"testing"
)

// pngMagic is the 8-byte PNG file signature. Kitty's graphics protocol accepts
// only PNG (f=100) or raw RGB/RGBA, so transmitted data must start with this.
var pngMagic = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}

func encodeGIF(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewPaletted(image.Rect(0, 0, w, h), color.Palette{color.Black, color.White})
	var buf bytes.Buffer
	if err := gif.Encode(&buf, img, nil); err != nil {
		t.Fatalf("encode gif: %v", err)
	}
	return buf.Bytes()
}

func encodeJPEG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatalf("encode jpeg: %v", err)
	}
	return buf.Bytes()
}

func encodePNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

// TestPNGData_NonPNGWithinTargetReencodesToPNG reproduces the blank-image bug:
// a GIF whose native size is within the transmit target skips the resize step,
// so before the fix its raw GIF bytes were handed to the terminal under an
// f=100 (PNG) header — Kitty/Ghostty fail to PNG-decode GIF data and draw
// nothing. PNGData must always return PNG-encoded bytes regardless of size.
func TestPNGData_NonPNGWithinTargetReencodesToPNG(t *testing.T) {
	cases := []struct {
		name string
		data []byte
	}{
		{"gif", encodeGIF(t, 20, 10)},
		{"jpeg", encodeJPEG(t, 20, 10)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			info, err := decodeImageInfo(tc.data)
			if err != nil {
				t.Fatalf("decode %s: %v", tc.name, err)
			}
			// target larger than the image => no downscale needed; the bug path.
			out, err := info.PNGData(info.Width*4, info.Height*4)
			if err != nil {
				t.Fatalf("PNGData: %v", err)
			}
			if !bytes.HasPrefix(out, pngMagic) {
				t.Fatalf("expected PNG output for %s within target, got %d bytes with prefix %x",
					tc.name, len(out), out[:min(8, len(out))])
			}
		})
	}
}

// TestPNGData_PNGWithinTargetPassesThrough keeps the fast path: an already-PNG
// image within the transmit target is returned untouched (no needless re-encode).
func TestPNGData_PNGWithinTargetPassesThrough(t *testing.T) {
	src := encodePNG(t, 20, 10)
	info, err := decodeImageInfo(src)
	if err != nil {
		t.Fatalf("decode png: %v", err)
	}
	out, err := info.PNGData(info.Width*4, info.Height*4)
	if err != nil {
		t.Fatalf("PNGData: %v", err)
	}
	if !bytes.Equal(out, src) {
		t.Fatalf("expected PNG within target to pass through unchanged")
	}
}

// TestPNGData_OversizedDownscalesToPNG preserves the existing downscale behaviour
// and guarantees PNG output for oversized non-PNG sources too.
func TestPNGData_OversizedDownscalesToPNG(t *testing.T) {
	info, err := decodeImageInfo(encodeGIF(t, 100, 100))
	if err != nil {
		t.Fatalf("decode gif: %v", err)
	}
	out, err := info.PNGData(20, 20)
	if err != nil {
		t.Fatalf("PNGData: %v", err)
	}
	if !bytes.HasPrefix(out, pngMagic) {
		t.Fatalf("expected PNG output for oversized gif")
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("decode resized: %v", err)
	}
	if cfg.Width != 20 || cfg.Height != 20 {
		t.Fatalf("expected 20x20 downscale, got %dx%d", cfg.Width, cfg.Height)
	}
}
