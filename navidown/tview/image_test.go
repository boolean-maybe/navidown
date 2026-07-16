package tview

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	nav "github.com/boolean-maybe/navidown/navidown"
)

func testPNGBytes() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, image.Black)
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func TestImageManager_PreResolveMarkdown(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pic.png"), testPNGBytes(), 0644); err != nil {
		t.Fatal(err)
	}
	sourceFile := filepath.Join(dir, "doc.md")

	resolver := nav.NewImageResolver([]string{dir})
	mgr := NewImageManager(resolver, 8, 16)

	var mu sync.Mutex
	var lastDone, lastTotal, calls int
	mgr.SetProgressCallback(func(done, total int) {
		mu.Lock()
		defer mu.Unlock()
		calls++
		lastDone, lastTotal = done, total
	})

	// pre-resolving raw markdown must warm the resolver cache and fire the
	// progress callback, without touching any widget.
	mgr.PreResolveMarkdown("![pic](pic.png)\n", sourceFile, 80, nil, nil)

	mu.Lock()
	gotCalls, gotDone, gotTotal := calls, lastDone, lastTotal
	mu.Unlock()
	if gotCalls != 1 || gotDone != 1 || gotTotal != 1 {
		t.Fatalf("callback fired calls=%d last=(%d/%d), want 1 (1/1)", gotCalls, gotDone, gotTotal)
	}

	// the image must now be a cache hit
	info, err := resolver.Resolve("pic.png", sourceFile)
	if err != nil || info == nil {
		t.Fatalf("image not cached after PreResolveMarkdown: info=%v err=%v", info, err)
	}
}

// writeFakeDot writes a shell script that behaves like `dot`: it copies a PNG
// fixture to the path given after -o. Returns the script path. Skips on Windows.
func writeFakeDot(t *testing.T, dir string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake dot script is POSIX-only")
	}
	fixture := filepath.Join(dir, "fixture.png")
	if err := os.WriteFile(fixture, testPNGBytes(), 0644); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(dir, "fake-dot")
	body := "#!/bin/sh\n" +
		"out=\"\"\n" +
		"while [ $# -gt 0 ]; do\n" +
		"  case \"$1\" in\n" +
		"    -o) shift; out=\"$1\" ;;\n" +
		"    -o*) out=\"${1#-o}\" ;;\n" +
		"  esac\n" +
		"  shift\n" +
		"done\n" +
		"cp \"" + fixture + "\" \"$out\"\n"
	if err := os.WriteFile(script, []byte(body), 0755); err != nil { //nolint:gosec // test fixture
		t.Fatal(err)
	}
	return script
}

// TestImageManager_PreResolveMarkdown_RendersDiagram proves PreResolveMarkdown
// pre-renders diagram blocks off the widget when GraphvizOptions are supplied,
// so the expensive dot subprocess runs during the off-thread pre-resolve.
func TestImageManager_PreResolveMarkdown_RendersDiagram(t *testing.T) {
	dir := t.TempDir()
	dotPath := writeFakeDot(t, dir)
	cacheDir := t.TempDir()

	resolver := nav.NewImageResolver([]string{dir})
	mgr := NewImageManager(resolver, 8, 16)

	md := "```dot\ndigraph { a -> b }\n```\n"
	mgr.PreResolveMarkdown(md, filepath.Join(dir, "doc.md"), 80, nil,
		&nav.GraphvizOptions{DotPath: dotPath, CacheDir: cacheDir})

	// the diagram must have been rendered to a PNG in the graphviz cache dir.
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		t.Fatal(err)
	}
	pngs := 0
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".png" {
			pngs++
		}
	}
	if pngs == 0 {
		t.Fatalf("PreResolveMarkdown did not pre-render the dot diagram (no PNG in cache %s)", cacheDir)
	}
}

func TestImageManager_PreResolveMarkdown_NoImages(t *testing.T) {
	resolver := nav.NewImageResolver([]string{t.TempDir()})
	mgr := NewImageManager(resolver, 8, 16)
	fired := false
	mgr.SetProgressCallback(func(int, int) { fired = true })

	// markdown with no images must not fire progress.
	mgr.PreResolveMarkdown("# just text\n\nno pictures here\n", "", 80, nil, nil)
	if fired {
		t.Fatal("progress callback fired for markdown with no images")
	}
}

func TestBuildPlaceholderLines(t *testing.T) {
	placeholder := &nav.ImagePlaceholder{
		ImageID: 1,
		Cols:    3,
		Rows:    2,
		URL:     "test.png",
	}

	lines := BuildPlaceholderLines(placeholder)

	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	for i, line := range lines {
		// Each line should start with a color tag and end with a reset tag
		if !strings.HasPrefix(line, "[#000001]") {
			t.Errorf("line %d: missing color tag prefix, got %q", i, line[:min(20, len(line))])
		}
		if !strings.HasSuffix(line, "[-]") {
			t.Errorf("line %d: missing reset tag suffix", i)
		}

		// Each line should contain placeholder runes
		if !strings.ContainsRune(line, placeholderRune) {
			t.Errorf("line %d: missing placeholder rune U+10EEEE", i)
		}

		// Count placeholder runes (should be 3 per line for 3 cols)
		count := strings.Count(line, string(placeholderRune))
		if count != 3 {
			t.Errorf("line %d: expected 3 placeholder runes, got %d", i, count)
		}
	}
}

func TestBuildPlaceholderLines_Zero(t *testing.T) {
	placeholder := &nav.ImagePlaceholder{
		ImageID: 1,
		Cols:    0,
		Rows:    0,
	}

	lines := BuildPlaceholderLines(placeholder)
	if lines != nil {
		t.Errorf("expected nil for zero dimensions, got %d lines", len(lines))
	}
}

func TestBuildPlaceholderLines_ColorEncoding(t *testing.T) {
	placeholder := &nav.ImagePlaceholder{
		ImageID: 0x00123456,
		Cols:    1,
		Rows:    1,
	}

	lines := BuildPlaceholderLines(placeholder)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}

	// Lower 3 bytes encode in the color tag
	if !strings.HasPrefix(lines[0], "[#123456]") {
		t.Errorf("expected color tag [#123456], got %q", lines[0][:min(15, len(lines[0]))])
	}
}

func TestBuildPlaceholderLines_IDDiacritic(t *testing.T) {
	// ID with non-zero upper byte should include a 3rd diacritic per cell
	placeholder := &nav.ImagePlaceholder{
		ImageID: 0x01010101,
		Cols:    2,
		Rows:    1,
	}

	lines := BuildPlaceholderLines(placeholder)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}

	// Upper byte = 1, so 3rd diacritic should be rowColumnDiacritics[1] = 0x030D
	runes := []rune(lines[0])
	// Skip color tag "[#010101]" = 9 runes
	// First cell: placeholder + row_diac + col_diac + id_diac
	// Find the first placeholder rune
	foundIDDiac := false
	for i, r := range runes {
		if r == 0x030D && i > 0 {
			// Check this isn't a row/col diac by verifying position
			foundIDDiac = true
			break
		}
	}
	if !foundIDDiac {
		t.Error("expected 3rd diacritic U+030D for upper byte 1")
	}

	// ID with zero upper byte should NOT include the 3rd diacritic
	placeholder2 := &nav.ImagePlaceholder{
		ImageID: 0x00010101,
		Cols:    2,
		Rows:    1,
	}
	lines2 := BuildPlaceholderLines(placeholder2)
	line1Runes := len([]rune(lines[0]))
	line2Runes := len([]rune(lines2[0]))
	// With 3rd diacritic: 2 more runes (1 per cell)
	if line1Runes != line2Runes+2 {
		t.Errorf("expected line with ID diacritic to have 2 more runes, got %d vs %d", line1Runes, line2Runes)
	}
}

func TestDiacriticsTable(t *testing.T) {
	// Verify the diacritics table has 297 entries (matching Kitty's gen/rowcolumn-diacritics.txt)
	if len(rowColumnDiacritics) != 297 {
		t.Errorf("expected 297 diacritics, got %d", len(rowColumnDiacritics))
	}

	// Verify first few entries match Kitty spec
	expected := []rune{0x0305, 0x030D, 0x030E, 0x0310, 0x0312}
	for i, want := range expected {
		if rowColumnDiacritics[i] != want {
			t.Errorf("diacritics[%d]: got U+%04X, want U+%04X", i, rowColumnDiacritics[i], want)
		}
	}
}

func TestKittyImageProcessor_NoTokens(t *testing.T) {
	// Use nil manager since no tokens means no resolution needed
	processor := &KittyImageProcessor{}

	lines := []string{"line one", "line two"}
	result := processor.ProcessImageTokens(lines, "", 80)

	if len(result) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(result))
	}
	if result[0] != "line one" || result[1] != "line two" {
		t.Error("lines should pass through unchanged when no tokens present")
	}
}
