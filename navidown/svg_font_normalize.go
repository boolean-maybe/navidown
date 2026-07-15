package navidown

import (
	"regexp"
	"strings"
)

// the embedded resvg wasm module registers exactly one font family:
// DejaVu Sans, aliased to the generic "sans-serif". it does NOT register
// "serif" or "monospace", and usvg does not fall back to the default family
// for a font-family list that names only unregistered families. so any text
// whose resolved font-family is e.g. "monospace" or "Consolas, monospace"
// renders as blank glyphs. see WasmRasterizer docs.
//
// fontFamilyPattern matches font-family declarations in both XML attribute
// form (font-family="...") and CSS form (font-family:...), inline or in a
// <style> block.
var fontFamilyPattern = regexp.MustCompile(`font-family\s*[:=]\s*("[^"]*"|'[^']*'|[^;"'}]*)`)

// normalizeSVGFonts appends "sans-serif" as a final fallback to every
// font-family declaration that does not already resolve to a family the
// embedded rasterizer can render. this guarantees text using "serif",
// "monospace", or an unregistered named family (e.g. Consolas, Segoe UI)
// still renders in DejaVu Sans rather than vanishing.
func normalizeSVGFonts(data []byte) []byte {
	s := string(data)
	result := fontFamilyPattern.ReplaceAllStringFunc(s, func(match string) string {
		if hasRenderableFamily(match) {
			return match
		}
		return appendSansSerifFallback(match)
	})
	return []byte(result)
}

// hasRenderableFamily reports whether the font-family declaration already
// names a family the rasterizer registers (sans-serif or DejaVu Sans),
// in which case no fallback is needed.
func hasRenderableFamily(decl string) bool {
	lower := strings.ToLower(decl)
	return strings.Contains(lower, "sans-serif") || strings.Contains(lower, "dejavu sans")
}

// appendSansSerifFallback inserts ", sans-serif" before the closing quote of
// a quoted value, or at the end of an unquoted value.
func appendSansSerifFallback(decl string) string {
	if strings.HasSuffix(decl, `"`) {
		return decl[:len(decl)-1] + `, sans-serif"`
	}
	if strings.HasSuffix(decl, `'`) {
		return decl[:len(decl)-1] + `, sans-serif'`
	}
	return decl + ", sans-serif"
}
