package navidown

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// LineCleaner returns a version of a rendered line without non-visible decoration.
// Implementations should remove things like ANSI escape sequences or UI markup tags.
type LineCleaner interface {
	Clean(line string) string
}

// LineCleanerFunc is a functional adapter for LineCleaner.
type LineCleanerFunc func(string) string

func (f LineCleanerFunc) Clean(line string) string { return f(line) }

// PositionCorrelator maps parsed markdown elements to their positions in rendered output.
type PositionCorrelator interface {
	// CorrelatePosition finds the best position match for an element in rendered lines.
	// Returns (lineIdx, startCol, endCol, found).
	//
	// startCol/endCol are rune columns in the cleaned line.
	CorrelatePosition(elem *NavElement, renderedLines []string, cleaner LineCleaner) (int, int, int, bool)
}

type matchCandidate struct {
	lineIdx  int
	score    int
	startCol int
	endCol   int
}

// ScoringCorrelator implements PositionCorrelator using a scoring-based best-match algorithm.
type ScoringCorrelator struct{}

func NewScoringCorrelator() *ScoringCorrelator { return &ScoringCorrelator{} }

// isWordBoundaryRunes checks if position is at a word boundary using pre-converted runes.
func (sc *ScoringCorrelator) isWordBoundaryRunes(runes []rune, pos int) bool {
	if pos < 0 || pos > len(runes) {
		return true
	}
	if pos == 0 || pos == len(runes) {
		return true
	}
	r := runes[pos]
	return unicode.IsSpace(r) || r == '•' || r == '.' || r == ',' ||
		r == ':' || r == ';' || r == '!' || r == '?' || r == '#'
}

// scoreMatchRunes scores a match using pre-converted runes to avoid repeated allocations.
func (sc *ScoringCorrelator) scoreMatchRunes(runes []rune, startCol, endCol int, elemType NavElementType, elemLevel int, cleanLine string) int {
	score := 0

	startBoundary := sc.isWordBoundaryRunes(runes, startCol)
	endBoundary := sc.isWordBoundaryRunes(runes, endCol)

	if startBoundary && endBoundary {
		score += 100
	} else if startBoundary || endBoundary {
		score += 50
	} else {
		inWord := false
		if startCol > 0 && startCol < len(runes) {
			prevChar := runes[startCol-1]
			if unicode.IsLetter(prevChar) || unicode.IsDigit(prevChar) {
				inWord = true
			}
		}
		if endCol > 0 && endCol < len(runes) && inWord {
			nextChar := runes[endCol]
			if unicode.IsLetter(nextChar) || unicode.IsDigit(nextChar) {
				return 0
			}
		}
		score += 25
	}

	if elemType == NavElementHeader {
		if strings.Contains(cleanLine, "##") {
			score += 50
			headerPrefix := strings.Repeat("#", elemLevel)
			if strings.Contains(cleanLine, headerPrefix) {
				score += 10
			}
		}
	}

	if startCol == 0 {
		score += 30
	} else if startCol < 15 {
		score += 10
	} else if startCol < 30 {
		score += 5
	}

	matchLen := endCol - startCol
	if matchLen >= 15 {
		score += 50
	} else if matchLen >= 10 {
		score += 30
	} else if matchLen >= 5 {
		score += 10
	}

	return score
}

func (sc *ScoringCorrelator) findCandidateMatches(elemText string, cleanLine string, lineIdx int, elemType NavElementType, elemLevel int) []matchCandidate {
	var candidates []matchCandidate

	// Convert to runes ONCE per line, reuse for all matches
	cleanRunes := []rune(cleanLine)
	elemRuneLen := utf8.RuneCountInString(elemText)

	searchOffset := 0
	for {
		byteIdx := strings.Index(cleanLine[searchOffset:], elemText)
		if byteIdx < 0 {
			break
		}
		actualByteIdx := searchOffset + byteIdx

		// Count runes up to byte position (still needed but we use utf8 directly)
		runeIdx := utf8.RuneCountInString(cleanLine[:actualByteIdx])
		endRuneIdx := runeIdx + elemRuneLen

		score := sc.scoreMatchRunes(cleanRunes, runeIdx, endRuneIdx, elemType, elemLevel, cleanLine)
		if score > 0 {
			candidates = append(candidates, matchCandidate{
				lineIdx:  lineIdx,
				score:    score,
				startCol: runeIdx,
				endCol:   endRuneIdx,
			})
		}

		searchOffset = actualByteIdx + len(elemText)
		if searchOffset >= len(cleanLine) {
			break
		}
	}

	return candidates
}

func (sc *ScoringCorrelator) CorrelatePosition(elem *NavElement, renderedLines []string, cleaner LineCleaner) (int, int, int, bool) {
	elemText := strings.TrimSpace(elem.Text)
	if elemText == "" {
		return 0, 0, 0, false
	}

	if cleaner == nil {
		cleaner = LineCleanerFunc(func(s string) string { return s })
	}

	var allCandidates []matchCandidate
	for lineIdx, line := range renderedLines {
		cleanLine := cleaner.Clean(line)
		candidates := sc.findCandidateMatches(elemText, cleanLine, lineIdx, elem.Type, elem.Level)
		allCandidates = append(allCandidates, candidates...)
	}

	if len(allCandidates) == 0 {
		return 0, 0, 0, false
	}

	bestCandidate := allCandidates[0]
	for _, candidate := range allCandidates[1:] {
		if candidate.score > bestCandidate.score {
			bestCandidate = candidate
		} else if candidate.score == bestCandidate.score && candidate.lineIdx < bestCandidate.lineIdx {
			bestCandidate = candidate
		}
	}

	return bestCandidate.lineIdx, bestCandidate.startCol, bestCandidate.endCol, true
}

// SimpleSubstringCorrelator implements PositionCorrelator using simple substring matching.
type SimpleSubstringCorrelator struct{}

func NewSimpleSubstringCorrelator() *SimpleSubstringCorrelator { return &SimpleSubstringCorrelator{} }

func (ssc *SimpleSubstringCorrelator) CorrelatePosition(elem *NavElement, renderedLines []string, cleaner LineCleaner) (int, int, int, bool) {
	elemText := strings.TrimSpace(elem.Text)
	if elemText == "" {
		return 0, 0, 0, false
	}

	if cleaner == nil {
		cleaner = LineCleanerFunc(func(s string) string { return s })
	}

	// Pre-compute element rune length once
	elemRuneLen := utf8.RuneCountInString(elemText)

	for lineIdx, line := range renderedLines {
		cleanLine := cleaner.Clean(line)
		byteIdx := strings.Index(cleanLine, elemText)
		if byteIdx >= 0 {
			runeIdx := utf8.RuneCountInString(cleanLine[:byteIdx])
			return lineIdx, runeIdx, runeIdx + elemRuneLen, true
		}
	}
	return 0, 0, 0, false
}
