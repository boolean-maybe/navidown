package navidown

import (
	"regexp"
	"strings"
	"sync"
)

// DiagramRenderer renders diagram source code to an image file and returns its path.
type DiagramRenderer interface {
	RenderToFile(source string) (string, error)
}

// diagramBlock represents a parsed fenced code block from markdown.
type diagramBlock struct {
	source    string
	openLine  int // first line of the opening fence
	closeLine int // line after closing fence (exclusive)
}

// extractDiagramBlocks scans markdown lines for fenced code blocks matching
// fenceRe and returns the split lines along with the identified blocks.
func extractDiagramBlocks(markdown string, fenceRe *regexp.Regexp) ([]string, []diagramBlock) {
	lines := strings.Split(markdown, "\n")
	var blocks []diagramBlock

	i := 0
	for i < len(lines) {
		match := fenceRe.FindStringSubmatch(lines[i])
		if match == nil {
			i++
			continue
		}

		fencePrefix := match[1]
		backticks := strings.TrimLeft(fencePrefix, " \t")
		fenceLen := len(backticks)
		openLine := i
		i++

		var source strings.Builder
		closed := false
		for i < len(lines) {
			trimmed := strings.TrimLeft(lines[i], " \t")
			if len(trimmed) >= fenceLen && strings.TrimRight(trimmed, "`") == "" && countLeadingBackticks(trimmed) >= fenceLen {
				i++
				closed = true
				break
			}
			source.WriteString(lines[i])
			source.WriteByte('\n')
			i++
		}

		if closed {
			blocks = append(blocks, diagramBlock{
				source:    source.String(),
				openLine:  openLine,
				closeLine: i,
			})
		}
	}

	return lines, blocks
}

// renderDiagramBlocks renders all blocks in parallel and returns a map of
// block index → image path. Missing entries indicate render errors.
func renderDiagramBlocks(blocks []diagramBlock, renderer DiagramRenderer) map[int]string {
	results := make(map[int]string)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for idx, block := range blocks {
		wg.Add(1)
		go func(idx int, source string) {
			defer wg.Done()
			path, err := renderer.RenderToFile(source)
			if err != nil {
				return
			}
			mu.Lock()
			results[idx] = path
			mu.Unlock()
		}(idx, block.source)
	}

	wg.Wait()
	return results
}

// reassembleDiagram rebuilds markdown from lines, substituting rendered blocks
// with image syntax using the given altText. Blocks missing from the rendered
// map are preserved as original fenced code.
func reassembleDiagram(lines []string, blocks []diagramBlock, rendered map[int]string, altText string) string {
	var result strings.Builder
	result.Grow(len(lines) * 40)

	blockIdx := 0
	i := 0
	for i < len(lines) {
		if blockIdx < len(blocks) && i == blocks[blockIdx].openLine {
			block := blocks[blockIdx]
			if pngPath, ok := rendered[blockIdx]; ok {
				result.WriteString("![" + altText + "](" + pngPath + ")")
				if block.closeLine < len(lines) {
					result.WriteByte('\n')
				}
			} else {
				for j := block.openLine; j < block.closeLine; j++ {
					result.WriteString(lines[j])
					if j < len(lines)-1 {
						result.WriteByte('\n')
					}
				}
			}
			i = block.closeLine
			blockIdx++
			continue
		}

		result.WriteString(lines[i])
		if i < len(lines)-1 {
			result.WriteByte('\n')
		}
		i++
	}

	return result.String()
}

// countLeadingBackticks returns the number of leading backtick characters in s.
func countLeadingBackticks(s string) int {
	count := 0
	for _, ch := range s {
		if ch == '`' {
			count++
		} else {
			break
		}
	}
	return count
}
