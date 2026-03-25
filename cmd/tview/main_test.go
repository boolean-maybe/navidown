package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsImageFile(t *testing.T) {
	supported := []string{
		"photo.png", "photo.jpg", "photo.jpeg", "photo.gif",
		"photo.bmp", "photo.tiff", "photo.tif", "photo.webp", "photo.svg",
	}
	for _, name := range supported {
		if !isImageFile(name) {
			t.Errorf("expected %q to be detected as image", name)
		}
	}

	// mixed case
	for _, name := range []string{"PHOTO.PNG", "Photo.Jpg", "image.SVG"} {
		if !isImageFile(name) {
			t.Errorf("expected %q (mixed case) to be detected as image", name)
		}
	}

	// non-image
	for _, name := range []string{"readme.md", "main.go", "data.json", "file.txt", "noext"} {
		if isImageFile(name) {
			t.Errorf("expected %q to NOT be detected as image", name)
		}
	}
}

func TestLoadContentImage(t *testing.T) {
	// create a temp PNG file (1x1 white pixel)
	dir := t.TempDir()
	imgPath := filepath.Join(dir, "test.png")
	// minimal valid PNG (1x1 pixel)
	pngData := []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xde, 0x00, 0x00, 0x00, 0x0c, 0x49, 0x44, 0x41,
		0x54, 0x08, 0xd7, 0x63, 0xf8, 0xcf, 0xc0, 0x00,
		0x00, 0x00, 0x02, 0x00, 0x01, 0xe2, 0x21, 0xbc,
		0x33, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e,
		0x44, 0xae, 0x42, 0x60, 0x82,
	}
	if err := os.WriteFile(imgPath, pngData, 0644); err != nil {
		t.Fatal(err)
	}

	content, sourcePath, err := loadContent(imgPath)
	if err != nil {
		t.Fatalf("loadContent returned error: %v", err)
	}

	if sourcePath != imgPath {
		t.Errorf("sourcePath = %q, want %q", sourcePath, imgPath)
	}

	if !strings.Contains(content, "![test.png]") {
		t.Errorf("expected synthetic markdown with image reference, got: %q", content)
	}
	if !strings.Contains(content, imgPath) {
		t.Errorf("expected absolute path in markdown, got: %q", content)
	}
}

func TestLoadContentMarkdown(t *testing.T) {
	dir := t.TempDir()
	mdPath := filepath.Join(dir, "test.md")
	mdContent := "# Hello\n"
	if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
		t.Fatal(err)
	}

	content, _, err := loadContent(mdPath)
	if err != nil {
		t.Fatalf("loadContent returned error: %v", err)
	}

	if content != mdContent {
		t.Errorf("expected raw markdown content %q, got %q", mdContent, content)
	}
}
