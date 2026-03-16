package navidown

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// MermaidOptions configures mermaid diagram rendering.
type MermaidOptions struct {
	MmdcPath        string        // path to mmdc binary; "" = lookup "mmdc" in PATH
	Theme           string        // mermaid theme; "" = "dark"
	BackgroundColor string        // background color; "" = "transparent"
	Scale           int           // render scale; 0 = 2 (retina)
	Timeout         time.Duration // render timeout; 0 = 30s
	CacheDir        string        // persistent cache dir; "" = auto (os.UserCacheDir()/navidown/mermaid)
}

func (o *MermaidOptions) resolvedTheme() string {
	if o.Theme != "" {
		return o.Theme
	}
	return "dark"
}

func (o *MermaidOptions) resolvedBackground() string {
	if o.BackgroundColor != "" {
		return o.BackgroundColor
	}
	return "transparent"
}

func (o *MermaidOptions) resolvedScale() int {
	if o.Scale > 0 {
		return o.Scale
	}
	return 2
}

func (o *MermaidOptions) resolvedTimeout() time.Duration {
	if o.Timeout > 0 {
		return o.Timeout
	}
	return 30 * time.Second
}

// MermaidRenderer renders mermaid source code to PNG files using mmdc.
type MermaidRenderer struct {
	opts          MermaidOptions
	cache         sync.Map // cache key hex -> absolute PNG path
	persistentDir string   // persistent cache (never deleted by Close)
	tempDir       string   // temp fallback (deleted by Close); "" if persistent worked
	workDir       string   // whichever dir is actually used
	mmdcPath      string
}

// NewMermaidRenderer creates a new renderer. Returns nil if mmdc is not found.
func NewMermaidRenderer(opts MermaidOptions) *MermaidRenderer {
	mmdcPath := opts.MmdcPath
	if mmdcPath == "" {
		resolved, err := exec.LookPath("mmdc")
		if err != nil {
			return nil
		}
		mmdcPath = resolved
	}

	persistentDir, tempDir, workDir := resolveCacheDir(opts.CacheDir)
	if workDir == "" {
		return nil
	}

	return &MermaidRenderer{
		opts:          opts,
		persistentDir: persistentDir,
		tempDir:       tempDir,
		workDir:       workDir,
		mmdcPath:      mmdcPath,
	}
}

// resolveCacheDir determines which directory to use for mermaid PNG caching.
// It tries (in order): explicit path, os.UserCacheDir()/navidown/mermaid, temp dir.
// Returns (persistentDir, tempDir, workDir). workDir is the dir to actually use.
func resolveCacheDir(explicit string) (persistentDir, tempDir, workDir string) {
	if explicit != "" {
		if err := os.MkdirAll(explicit, 0700); err == nil {
			return explicit, "", explicit
		}
	}

	if ucd, err := os.UserCacheDir(); err == nil {
		dir := filepath.Join(ucd, "navidown", "mermaid")
		if err := os.MkdirAll(dir, 0700); err == nil {
			return dir, "", dir
		}
	}

	if td, err := os.MkdirTemp("", "navidown-mermaid-"); err == nil {
		return "", td, td
	}

	return "", "", ""
}

// cacheKey computes a hash incorporating the mermaid source and render options
// (theme, background, scale) so that option changes don't produce stale hits.
func (r *MermaidRenderer) cacheKey(source string) string {
	h := sha256.New()
	h.Write([]byte(source))
	h.Write([]byte{0}) // separator
	h.Write([]byte(r.opts.resolvedTheme()))
	h.Write([]byte{0})
	h.Write([]byte(r.opts.resolvedBackground()))
	h.Write([]byte{0})
	_, _ = fmt.Fprintf(h, "%d", r.opts.resolvedScale())
	return fmt.Sprintf("%x", h.Sum(nil))
}

// RenderToFile renders mermaid source to a PNG file and returns its absolute path.
// Results are cached by content hash (in-memory and on disk).
func (r *MermaidRenderer) RenderToFile(source string) (string, error) {
	key := r.cacheKey(source)
	outputPath := filepath.Join(r.workDir, key+".png")

	// check in-memory cache
	if cached, ok := r.cache.Load(key); ok {
		if path, ok := cached.(string); ok {
			return path, nil
		}
	}

	// check disk cache
	if _, err := os.Stat(outputPath); err == nil {
		r.cache.Store(key, outputPath)
		return outputPath, nil
	}

	inputPath := filepath.Join(r.workDir, key+".mmd")

	if err := os.WriteFile(inputPath, []byte(source), 0600); err != nil {
		return "", fmt.Errorf("write mermaid source: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), r.opts.resolvedTimeout())
	defer cancel()

	cmd := exec.CommandContext(ctx, r.mmdcPath, // #nosec G204 -- mmdcPath from LookPath("mmdc") or user-provided
		"-i", inputPath,
		"-o", outputPath,
		"-t", r.opts.resolvedTheme(),
		"-b", r.opts.resolvedBackground(),
		"-s", fmt.Sprintf("%d", r.opts.resolvedScale()),
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("mmdc failed: %w\n%s", err, string(out))
	}

	// clean up .mmd input file
	_ = os.Remove(inputPath)

	r.cache.Store(key, outputPath)
	return outputPath, nil
}

// Close releases resources. Only removes the temp dir (if used as fallback).
// Persistent cache directories are preserved for future sessions.
func (r *MermaidRenderer) Close() {
	if r.tempDir != "" {
		_ = os.RemoveAll(r.tempDir)
	}
}

// mermaidFenceRe matches the opening of a mermaid fenced code block.
var mermaidFenceRe = regexp.MustCompile("^(\\s*`{3,})mermaid\\s*$")

// mermaidBlock represents a parsed mermaid code block from markdown.
type mermaidBlock struct {
	source    string
	openLine  int // first line of ```mermaid fence
	closeLine int // line after closing fence (exclusive)
}

// extractMermaidBlocks scans markdown lines for ```mermaid fences and returns
// the split lines along with the identified blocks (source + positions).
func extractMermaidBlocks(markdown string) ([]string, []mermaidBlock) {
	lines := strings.Split(markdown, "\n")
	var blocks []mermaidBlock

	i := 0
	for i < len(lines) {
		match := mermaidFenceRe.FindStringSubmatch(lines[i])
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
			blocks = append(blocks, mermaidBlock{
				source:    source.String(),
				openLine:  openLine,
				closeLine: i,
			})
		}
		// unclosed fences are simply not collected — they stay as-is
	}

	return lines, blocks
}

// renderMermaidBlocks renders all blocks in parallel and returns a map of
// block index → PNG path. Missing entries indicate render errors.
func renderMermaidBlocks(blocks []mermaidBlock, renderer *MermaidRenderer) map[int]string {
	results := make(map[int]string)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for idx, block := range blocks {
		wg.Add(1)
		go func(idx int, source string) {
			defer wg.Done()
			pngPath, err := renderer.RenderToFile(source)
			if err != nil {
				return
			}
			mu.Lock()
			results[idx] = pngPath
			mu.Unlock()
		}(idx, block.source)
	}

	wg.Wait()
	return results
}

// reassembleMermaid rebuilds markdown from lines, substituting rendered blocks.
// Blocks missing from the rendered map are preserved as original fenced code.
func reassembleMermaid(lines []string, blocks []mermaidBlock, rendered map[int]string) string {
	var result strings.Builder
	result.Grow(len(lines) * 40) // rough estimate

	blockIdx := 0
	i := 0
	for i < len(lines) {
		// check if current line is the start of the next block
		if blockIdx < len(blocks) && i == blocks[blockIdx].openLine {
			block := blocks[blockIdx]
			if pngPath, ok := rendered[blockIdx]; ok {
				result.WriteString("![mermaid diagram](" + pngPath + ")")
				if block.closeLine < len(lines) {
					result.WriteByte('\n')
				}
			} else {
				// render error — preserve original lines
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

// preprocessMermaid scans raw markdown for ```mermaid blocks and replaces them
// with ![mermaid diagram](path.png) using the given renderer.
// Blocks are rendered in parallel. On render error, the original code block is preserved.
func preprocessMermaid(markdown string, renderer *MermaidRenderer) string {
	if renderer == nil {
		return markdown
	}

	lines, blocks := extractMermaidBlocks(markdown)
	if len(blocks) == 0 {
		return markdown
	}

	rendered := renderMermaidBlocks(blocks, renderer)
	return reassembleMermaid(lines, blocks, rendered)
}

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
