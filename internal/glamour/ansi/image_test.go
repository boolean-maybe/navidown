package ansi

import "testing"

func TestLooksLikeImage(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"photo.png", true},
		{"photo.jpg", true},
		{"photo.jpeg", true},
		{"photo.gif", true},
		{"photo.bmp", true},
		{"photo.webp", true},
		{"photo.tiff", true},
		{"photo.tif", true},
		{"diagram.svg", true},
		{"DIAGRAM.SVG", true},
		{"photo.PNG", true},
		{"photo.png?v=2", true},
		{"photo.png#anchor", true},
		{"readme.md", false},
		{"script.js", false},
		{"", false},
		{"noextension", false},
		{"https://example.com/image.svg?token=abc", true},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := looksLikeImage(tt.url)
			if got != tt.want {
				t.Errorf("looksLikeImage(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}
