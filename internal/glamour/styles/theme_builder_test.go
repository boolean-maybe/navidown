package styles

import (
	"testing"
)

func TestBuildStyleConfig_GruvboxDark(t *testing.T) {
	cfg := GruvboxDarkStyleConfig

	// document text
	if cfg.Document.Color == nil || *cfg.Document.Color != "#ebdbb2" {
		t.Errorf("Document.Color: got %v, want #ebdbb2", cfg.Document.Color)
	}

	// heading: purple, bold
	if cfg.Heading.Color == nil || *cfg.Heading.Color != "#d3869b" {
		t.Errorf("Heading.Color: got %v, want #d3869b", cfg.Heading.Color)
	}
	if cfg.Heading.Bold == nil || !*cfg.Heading.Bold {
		t.Error("Heading.Bold should be true")
	}

	// blockquote: yellow, italic, "│ " indent
	if cfg.BlockQuote.Color == nil || *cfg.BlockQuote.Color != "#fabd2f" {
		t.Errorf("BlockQuote.Color: got %v, want #fabd2f", cfg.BlockQuote.Color)
	}
	if cfg.BlockQuote.Italic == nil || !*cfg.BlockQuote.Italic {
		t.Error("BlockQuote should be italic")
	}
	if cfg.BlockQuote.IndentToken == nil || *cfg.BlockQuote.IndentToken != "│ " {
		t.Errorf("BlockQuote.IndentToken: got %v, want '│ '", cfg.BlockQuote.IndentToken)
	}

	// link: blue, underlined
	if cfg.Link.Color == nil || *cfg.Link.Color != "#83a598" {
		t.Errorf("Link.Color: got %v, want #83a598", cfg.Link.Color)
	}
	if cfg.Link.Underline == nil || !*cfg.Link.Underline {
		t.Error("Link should be underlined")
	}

	// inline code: green
	if cfg.Code.Color == nil || *cfg.Code.Color != "#b8bb26" {
		t.Errorf("Code.Color: got %v, want #b8bb26", cfg.Code.Color)
	}

	// chroma background
	if cfg.CodeBlock.Chroma == nil {
		t.Fatal("Chroma should not be nil")
	}
	if cfg.CodeBlock.Chroma.Background.BackgroundColor == nil || *cfg.CodeBlock.Chroma.Background.BackgroundColor != "#282828" {
		t.Errorf("Chroma.Background: got %v, want #282828", cfg.CodeBlock.Chroma.Background.BackgroundColor)
	}

	// chroma keyword
	if cfg.CodeBlock.Chroma.Keyword.Color == nil || *cfg.CodeBlock.Chroma.Keyword.Color != "#fb4934" {
		t.Errorf("Chroma.Keyword.Color: got %v, want #fb4934", cfg.CodeBlock.Chroma.Keyword.Color)
	}

	// optional ChromaNumber should be set
	if cfg.CodeBlock.Chroma.LiteralNumber.Color == nil || *cfg.CodeBlock.Chroma.LiteralNumber.Color != "#d3869b" {
		t.Errorf("Chroma.LiteralNumber.Color: got %v, want #d3869b", cfg.CodeBlock.Chroma.LiteralNumber.Color)
	}
}

func TestBuildStyleConfig_OptionalChromaNumber(t *testing.T) {
	// when ChromaNumber is empty, LiteralNumber should have no color
	cfg := BuildStyleConfig(ThemeColors{
		Fg: "#aaa", Bg: "#111", Muted: "#555",
		Yellow: "#ff0", Orange: "#f80", Purple: "#a0f",
		Green: "#0f0", Blue: "#00f", Cyan: "#0ff", Red: "#f00",
		ChromaKeyword: "#f00", ChromaNumber: "",
		ChromaStringEscape: "#f80", ChromaPreproc: "#0ff",
		ChromaTag: "#f00",
	})

	if cfg.CodeBlock.Chroma.LiteralNumber.Color != nil {
		t.Errorf("LiteralNumber.Color should be nil when ChromaNumber is empty, got %v", *cfg.CodeBlock.Chroma.LiteralNumber.Color)
	}
}

func TestBuildStyleConfig_StructuralConsistency(t *testing.T) {
	cfg := GruvboxDarkStyleConfig

	// H1-H6 prefixes
	prefixes := map[string]string{
		"H1": "# ", "H2": "## ", "H3": "### ",
		"H4": "#### ", "H5": "##### ", "H6": "###### ",
	}
	blocks := map[string]string{
		"H1": cfg.H1.Prefix, "H2": cfg.H2.Prefix,
		"H3": cfg.H3.Prefix, "H4": cfg.H4.Prefix,
		"H5": cfg.H5.Prefix, "H6": cfg.H6.Prefix,
	}
	for name, want := range prefixes {
		if got := blocks[name]; got != want {
			t.Errorf("%s.Prefix: got %q, want %q", name, got, want)
		}
	}

	// strikethrough crossed out
	if cfg.Strikethrough.CrossedOut == nil || !*cfg.Strikethrough.CrossedOut {
		t.Error("Strikethrough should have CrossedOut=true")
	}

	// task checkboxes
	if cfg.Task.Ticked != "[✓] " {
		t.Errorf("Task.Ticked: got %q, want %q", cfg.Task.Ticked, "[✓] ")
	}

	// definition description
	if cfg.DefinitionDescription.BlockPrefix != "\n🠶 " {
		t.Errorf("DefinitionDescription.BlockPrefix: got %q", cfg.DefinitionDescription.BlockPrefix)
	}

	// GenericEmph: italic, no color
	if cfg.CodeBlock.Chroma.GenericEmph.Italic == nil || !*cfg.CodeBlock.Chroma.GenericEmph.Italic {
		t.Error("GenericEmph should be italic")
	}
	if cfg.CodeBlock.Chroma.GenericEmph.Color != nil {
		t.Error("GenericEmph should not have a color")
	}

	// GenericStrong: bold, no color
	if cfg.CodeBlock.Chroma.GenericStrong.Bold == nil || !*cfg.CodeBlock.Chroma.GenericStrong.Bold {
		t.Error("GenericStrong should be bold")
	}
	if cfg.CodeBlock.Chroma.GenericStrong.Color != nil {
		t.Error("GenericStrong should not have a color")
	}
}
