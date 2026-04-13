package navidown

import (
	"os"
	"testing"

	"github.com/boolean-maybe/navidown/internal/glamour/styles"
)

func TestNewANSIRendererWithStyle(t *testing.T) {
	tests := []struct {
		name      string
		styleName string
		wantDark  bool
	}{
		{"dark style", "dark", true},
		{"light style", "light", false},
		{"unknown defaults to dark", "unknown", true},
		{"empty defaults to dark", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer := NewANSIRendererWithStyle(tt.styleName)
			if renderer == nil {
				t.Fatal("NewANSIRendererWithStyle returned nil")
			}

			// Check that renderer was created
			if renderer.wordWrap != 0 {
				t.Errorf("expected wordWrap=0, got %d", renderer.wordWrap)
			}

			// Verify margins are cleared
			if renderer.glamourStyle.Document.Margin != nil && *renderer.glamourStyle.Document.Margin != 0 {
				t.Errorf("Document.Margin should be 0, got %d", *renderer.glamourStyle.Document.Margin)
			}
			if renderer.glamourStyle.CodeBlock.Margin != nil && *renderer.glamourStyle.CodeBlock.Margin != 0 {
				t.Errorf("CodeBlock.Margin should be 0, got %d", *renderer.glamourStyle.CodeBlock.Margin)
			}
		})
	}
}

func TestDetectStyleFromEnvironment(t *testing.T) {
	tests := []struct {
		name        string
		colorfgbg   string
		wantDark    bool
		description string
	}{
		{"light background (15)", "0;15", false, "bg=15 >= 8 should use light"},
		{"dark background (0)", "15;0", true, "bg=0 < 8 should use dark"},
		{"light background (8)", "0;8", false, "bg=8 >= 8 should use light"},
		{"dark background (7)", "0;7", true, "bg=7 < 8 should use dark"},
		{"missing env", "", true, "missing env should default to dark"},
		{"invalid format", "invalid", true, "invalid format should default to dark"},
		{"single value", "15", true, "single value should default to dark"},
		{"extra semicolons", "0;1;15", false, "should use last component (15)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				if err := os.Unsetenv("COLORFGBG"); err != nil {
					t.Fatalf("unset COLORFGBG: %v", err)
				}
			})

			// Set environment variable
			if tt.colorfgbg == "" {
				if err := os.Unsetenv("COLORFGBG"); err != nil {
					t.Fatalf("unset COLORFGBG: %v", err)
				}
			} else {
				if err := os.Setenv("COLORFGBG", tt.colorfgbg); err != nil {
					t.Fatalf("set COLORFGBG: %v", err)
				}
			}

			style := detectStyleFromEnvironment()

			// Check if we got the right style config
			isDark := style.Document.Color == styles.DarkStyleConfig.Document.Color
			if isDark != tt.wantDark {
				t.Errorf("%s: got dark=%v, want dark=%v", tt.description, isDark, tt.wantDark)
			}
		})
	}
}

func TestWithCodeTheme(t *testing.T) {
	original := NewANSIRenderer()
	modified := original.WithCodeTheme("dracula")

	if modified.glamourStyle.CodeBlock.Theme != "dracula" {
		t.Errorf("expected Theme=dracula, got %q", modified.glamourStyle.CodeBlock.Theme)
	}
	if modified.glamourStyle.CodeBlock.Chroma != nil {
		t.Error("expected Chroma=nil after WithCodeTheme")
	}

	// should extract dracula's background color (#282a36)
	if modified.glamourStyle.CodeBlock.BackgroundColor == nil {
		t.Fatal("expected BackgroundColor to be set from chroma theme")
	}
	if *modified.glamourStyle.CodeBlock.BackgroundColor != "#282a36" {
		t.Errorf("expected BackgroundColor=#282a36, got %q", *modified.glamourStyle.CodeBlock.BackgroundColor)
	}

	// original unchanged
	if original.glamourStyle.CodeBlock.Theme == "dracula" {
		t.Error("WithCodeTheme mutated the original renderer")
	}
}

func TestWithCodeBackground(t *testing.T) {
	original := NewANSIRenderer()
	modified := original.WithCodeBackground("#282a36")

	if modified.glamourStyle.CodeBlock.BackgroundColor == nil || *modified.glamourStyle.CodeBlock.BackgroundColor != "#282a36" {
		t.Errorf("expected BackgroundColor=#282a36, got %v", modified.glamourStyle.CodeBlock.BackgroundColor)
	}

	// original unchanged
	if original.glamourStyle.CodeBlock.BackgroundColor != nil && *original.glamourStyle.CodeBlock.BackgroundColor == "#282a36" {
		t.Error("WithCodeBackground mutated the original renderer")
	}
}

func TestWithCodeBorder(t *testing.T) {
	original := NewANSIRenderer()
	modified := original.WithCodeBorder("#6272a4")

	if modified.glamourStyle.CodeBlock.Color == nil || *modified.glamourStyle.CodeBlock.Color != "#6272a4" {
		t.Errorf("expected Color=#6272a4, got %v", modified.glamourStyle.CodeBlock.Color)
	}

	// original unchanged
	if original.glamourStyle.CodeBlock.Color != nil && *original.glamourStyle.CodeBlock.Color == "#6272a4" {
		t.Error("WithCodeBorder mutated the original renderer")
	}
}

func TestWithCodeChaining(t *testing.T) {
	base := NewANSIRenderer().WithWordWrap(80)
	result := base.WithCodeTheme("monokai").WithCodeBackground("#1e1e1e").WithCodeBorder("#444")

	if result.glamourStyle.CodeBlock.Theme != "monokai" {
		t.Errorf("expected Theme=monokai, got %q", result.glamourStyle.CodeBlock.Theme)
	}
	if result.glamourStyle.CodeBlock.BackgroundColor == nil || *result.glamourStyle.CodeBlock.BackgroundColor != "#1e1e1e" {
		t.Error("expected BackgroundColor=#1e1e1e")
	}
	if result.glamourStyle.CodeBlock.Color == nil || *result.glamourStyle.CodeBlock.Color != "#444" {
		t.Error("expected Color=#444")
	}
	if result.wordWrap != 80 {
		t.Errorf("expected wordWrap=80 preserved through chain, got %d", result.wordWrap)
	}
}

func TestNewANSIRendererWithStyle_RegisteredStyle(t *testing.T) {
	renderer := NewANSIRendererWithStyle("dracula")
	if renderer == nil {
		t.Fatal("NewANSIRendererWithStyle returned nil for registered style")
	}

	// dracula should differ from dark in at least the document color
	dark := NewANSIRendererWithStyle("dark")
	if renderer.glamourStyle.Document.Color == dark.glamourStyle.Document.Color {
		t.Error("dracula style should differ from dark style")
	}
}

func TestNewANSIRendererWithStyle_AllRegisteredStyles(t *testing.T) {
	// every entry in DefaultStyles must produce a non-nil renderer
	// with distinct Document.Color from the fallback dark style
	dark := NewANSIRendererWithStyle("dark")

	for name := range styles.DefaultStyles {
		t.Run(name, func(t *testing.T) {
			r := NewANSIRendererWithStyle(name)
			if r == nil {
				t.Fatalf("NewANSIRendererWithStyle(%q) returned nil", name)
			}
			// margins should always be cleared
			if r.glamourStyle.Document.Margin != nil && *r.glamourStyle.Document.Margin != 0 {
				t.Errorf("Document.Margin should be 0, got %d", *r.glamourStyle.Document.Margin)
			}
			// non-dark/ascii/notty styles should have a distinct Document.Color
			if name != "dark" && name != "ascii" && name != "notty" {
				if r.glamourStyle.Document.Color != nil && dark.glamourStyle.Document.Color != nil &&
					*r.glamourStyle.Document.Color == *dark.glamourStyle.Document.Color {
					t.Errorf("style %q has same Document.Color as dark (%s)", name, *dark.glamourStyle.Document.Color)
				}
			}
		})
	}
}

func TestAllRegisteredStyles_RenderSmoke(t *testing.T) {
	const sample = "# Heading\n\nHello **world**. `code` and *italic*.\n\n- item\n- item\n\n```go\nfunc main() {}\n```\n"

	for name := range styles.DefaultStyles {
		t.Run(name, func(t *testing.T) {
			r := NewANSIRendererWithStyle(name)
			result, err := r.Render(sample)
			if err != nil {
				t.Fatalf("Render with style %q failed: %v", name, err)
			}
			if len(result.Lines) < 3 {
				t.Errorf("expected at least 3 output lines, got %d", len(result.Lines))
			}
		})
	}
}

func TestNewANSIRendererBackwardsCompatibility(t *testing.T) {
	// NewANSIRenderer should behave exactly like NewANSIRendererWithStyle("dark")
	r1 := NewANSIRenderer()
	r2 := NewANSIRendererWithStyle("dark")

	if r1.wordWrap != r2.wordWrap {
		t.Errorf("wordWrap mismatch: %d vs %d", r1.wordWrap, r2.wordWrap)
	}

	// Both should have margins cleared
	if r1.glamourStyle.Document.Margin != nil && *r1.glamourStyle.Document.Margin != 0 {
		t.Error("NewANSIRenderer should clear Document.Margin")
	}
	if r2.glamourStyle.Document.Margin != nil && *r2.glamourStyle.Document.Margin != 0 {
		t.Error("NewANSIRendererWithStyle('dark') should clear Document.Margin")
	}
}
