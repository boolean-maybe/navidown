package navidown

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestProfile_ParallelTimeline(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping profiling test in short mode")
	}

	delay := 500 * time.Millisecond
	nBlocks := 4

	var md strings.Builder
	md.WriteString("# Profiling test\n\n")
	for i := 0; i < nBlocks; i++ {
		fmt.Fprintf(&md, "```mermaid\ngraph TD\n    A%d-->B%d\n```\n\n", i, i)
	}
	markdown := md.String()

	cacheDir := t.TempDir()

	// use a wrapper around renderMermaidBlocks that records timestamps
	epoch := time.Now()
	type span struct {
		blockIdx int
		start    time.Duration
		end      time.Duration
	}
	var spans []span
	var spanMu sync.Mutex

	// we can't easily hook into renderMermaidBlocks, so instead we'll
	// wrap RenderToFile by timing it from the outside using a custom approach.
	// Simpler: just measure with the existing renderer and check total time.

	// create a slow mmdc renderer
	fixturePNG := minimalPNG()
	scriptDir := t.TempDir()
	scriptPath := filepath.Join(scriptDir, "slow-mmdc")
	sleepArg := fmt.Sprintf("%d.%03d", int(delay.Milliseconds())/1000, int(delay.Milliseconds())%1000)
	script := fmt.Sprintf("#!/bin/sh\nsleep %s\nwhile [ $# -gt 0 ]; do\n  case \"$1\" in\n    -o) shift; cp \"%s\" \"$1\" ;;\n  esac\n  shift\ndone\n",
		sleepArg, filepath.Join(scriptDir, "fixture.png"))
	if err := os.WriteFile(filepath.Join(scriptDir, "fixture.png"), fixturePNG, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}

	renderer := NewMermaidRenderer(MermaidOptions{MmdcPath: scriptPath, CacheDir: cacheDir})
	if renderer == nil {
		t.Fatal("NewMermaidRenderer returned nil")
	}
	t.Cleanup(renderer.Close)

	// manually call extractMermaidBlocks + parallel render with timestamps
	lines, blocks := extractMermaidBlocks(markdown)
	if len(blocks) != nBlocks {
		t.Fatalf("expected %d blocks, got %d", nBlocks, len(blocks))
	}

	results := make(map[int]string)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for idx, block := range blocks {
		wg.Add(1)
		go func(idx int, source string) {
			defer wg.Done()
			start := time.Since(epoch)
			pngPath, err := renderer.RenderToFile(source)
			end := time.Since(epoch)
			spanMu.Lock()
			spans = append(spans, span{blockIdx: idx, start: start, end: end})
			spanMu.Unlock()
			if err != nil {
				return
			}
			mu.Lock()
			results[idx] = pngPath
			mu.Unlock()
		}(idx, block.source)
	}
	wg.Wait()

	result := reassembleMermaid(lines, blocks, results)
	imgCount := strings.Count(result, "![mermaid diagram](")
	if imgCount != nBlocks {
		t.Fatalf("expected %d images, got %d", nBlocks, imgCount)
	}

	// print timeline
	t.Logf("")
	t.Logf("=== parallel execution timeline ===")
	for _, s := range spans {
		barStart := int(s.start.Milliseconds() / 10)
		barEnd := int(s.end.Milliseconds() / 10)
		bar := strings.Repeat(" ", barStart) + strings.Repeat("█", barEnd-barStart)
		t.Logf("block %d: %6dms - %6dms  %s", s.blockIdx, s.start.Milliseconds(), s.end.Milliseconds(), bar)
	}
	t.Logf("")

	// verify overlap: all blocks should start before the first one ends
	var maxStart, minEnd time.Duration
	for i, s := range spans {
		if i == 0 || s.start > maxStart {
			maxStart = s.start
		}
		if i == 0 || s.end < minEnd {
			minEnd = s.end
		}
	}
	if maxStart >= minEnd {
		t.Errorf("blocks are NOT overlapping: last start (%v) >= first end (%v)", maxStart, minEnd)
	} else {
		overlap := minEnd - maxStart
		t.Logf("confirmed overlap: all blocks running concurrently for at least %v", overlap)
	}

	totalTime := time.Duration(0)
	for _, s := range spans {
		if s.end > totalTime {
			totalTime = s.end
		}
	}
	sequentialEstimate := time.Duration(nBlocks) * delay
	t.Logf("wall clock: %v, sequential estimate: %v, speedup: %.1fx", totalTime, sequentialEstimate, float64(sequentialEstimate)/float64(totalTime))
}
