package navidown

// MarkerCorrelator implements PositionCorrelator using invisible markers
// injected during glamour rendering. This provides 100% reliable position
// detection regardless of duplicate text or styling.
//
// If no markers are found (e.g., when using a non-glamour renderer or
// mocked lines in tests), it falls back to ScoringCorrelator.
type MarkerCorrelator struct {
	// Cache of extracted marker positions for current document
	cachedPositions []MarkerPosition
	cachedLineCount int    // Number of lines when cache was created
	cachedFirstLine string // First line content for validation (empty if no lines)

	// Counters for matching elements to markers in document order
	linkCounter   int
	headerCounter map[int]int // level -> count

	// Fallback correlator for when markers aren't present
	fallback *ScoringCorrelator
}

// NewMarkerCorrelator creates a new marker-based correlator.
func NewMarkerCorrelator() *MarkerCorrelator {
	return &MarkerCorrelator{
		headerCounter: make(map[int]int),
		fallback:      NewScoringCorrelator(),
	}
}

// Reset clears the correlator state for a new document.
func (mc *MarkerCorrelator) Reset() {
	mc.cachedPositions = nil
	mc.cachedLineCount = 0
	mc.cachedFirstLine = ""
	mc.linkCounter = 0
	mc.headerCounter = make(map[int]int)
}

// CorrelatePosition finds the position of an element using marker extraction.
// Elements are matched to markers in document order.
// Falls back to ScoringCorrelator if no markers are found.
func (mc *MarkerCorrelator) CorrelatePosition(elem *NavElement, renderedLines []string, cleaner LineCleaner) (int, int, int, bool) {
	// Extract markers if not cached or lines changed
	if !mc.isCacheValid(renderedLines) {
		mc.cachedPositions = ExtractAllMarkers(renderedLines, cleaner)
		mc.cachedLineCount = len(renderedLines)
		if len(renderedLines) > 0 {
			mc.cachedFirstLine = renderedLines[0]
		} else {
			mc.cachedFirstLine = ""
		}
	}

	// If no markers were found, fall back to scoring correlator
	if len(mc.cachedPositions) == 0 {
		return mc.fallback.CorrelatePosition(elem, renderedLines, cleaner)
	}

	switch elem.Type {
	case NavElementURL:
		lineIdx, startCol, endCol, found := mc.correlateLinkPosition()
		if found {
			return lineIdx, startCol, endCol, true
		}
		// Fall back if marker not found for this specific element
		return mc.fallback.CorrelatePosition(elem, renderedLines, cleaner)
	case NavElementHeader:
		lineIdx, startCol, endCol, found := mc.correlateHeaderPosition(elem.Level)
		if found {
			return lineIdx, startCol, endCol, true
		}
		// Fall back if marker not found for this specific element
		return mc.fallback.CorrelatePosition(elem, renderedLines, cleaner)
	default:
		return 0, 0, 0, false
	}
}

// isCacheValid checks if the cached positions are still valid.
// Uses line count and first line content to detect changes reliably.
func (mc *MarkerCorrelator) isCacheValid(renderedLines []string) bool {
	if mc.cachedPositions == nil {
		return false
	}
	if len(renderedLines) != mc.cachedLineCount {
		return false
	}
	// Check first line content to catch content changes with same line count
	if len(renderedLines) > 0 {
		return renderedLines[0] == mc.cachedFirstLine
	}
	return mc.cachedFirstLine == ""
}

// correlateLinkPosition finds the next link marker position.
func (mc *MarkerCorrelator) correlateLinkPosition() (int, int, int, bool) {
	targetOccurrence := mc.linkCounter

	currentOccurrence := 0
	for _, pos := range mc.cachedPositions {
		if pos.Type == MarkerTypeLink {
			if currentOccurrence == targetOccurrence {
				mc.linkCounter++
				return pos.LineIdx, pos.StartCol, pos.EndCol, true
			}
			currentOccurrence++
		}
	}

	return 0, 0, 0, false
}

// correlateHeaderPosition finds the next header marker position for a given level.
func (mc *MarkerCorrelator) correlateHeaderPosition(level int) (int, int, int, bool) {
	targetOccurrence := mc.headerCounter[level]

	currentOccurrence := 0
	for _, pos := range mc.cachedPositions {
		if pos.Type == MarkerTypeHeader && pos.Level == level {
			if currentOccurrence == targetOccurrence {
				mc.headerCounter[level]++
				return pos.LineIdx, pos.StartCol, pos.EndCol, true
			}
			currentOccurrence++
		}
	}

	return 0, 0, 0, false
}
