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
