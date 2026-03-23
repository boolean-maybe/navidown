package tview

import (
	"strings"

	nav "github.com/boolean-maybe/navidown/navidown"
	"github.com/boolean-maybe/navidown/util"
	"github.com/rivo/tview"
)

// convertDisplayLines converts rendered lines (mix of ANSI and tview-formatted
// image placeholders) into tview-ready display lines.
//
// Image placeholder lines (containing the Kitty placeholder rune) already use
// tview tags and pass through with only marker stripping. All other lines are
// ANSI-escaped: literal bracket patterns like [link] are escaped so tview
// doesn't misinterpret them as color/style tags, then ANSI codes are converted
// to tview tags.
func convertDisplayLines(lines []string, converter *util.AnsiConverter) []string {
	result := make([]string, len(lines))
	for i, line := range lines {
		if strings.ContainsRune(line, placeholderRune) {
			result[i] = nav.StripMarkers(line)
		} else if converter != nil {
			result[i] = nav.StripMarkers(converter.Convert(line))
		} else {
			result[i] = nav.StripMarkers(tview.TranslateANSI(tview.Escape(line)))
		}
	}
	return result
}
