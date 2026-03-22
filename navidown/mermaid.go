package navidown

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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

	persistentDir, tempDir, workDir := resolveCacheDir(opts.CacheDir, "mermaid")
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

// extractMermaidBlocks scans markdown lines for ```mermaid fences and returns
// the split lines along with the identified blocks (source + positions).
func extractMermaidBlocks(markdown string) ([]string, []diagramBlock) {
	return extractDiagramBlocks(markdown, mermaidFenceRe)
}

// reassembleMermaid rebuilds markdown from lines, substituting rendered blocks.
// Blocks missing from the rendered map are preserved as original fenced code.
func reassembleMermaid(lines []string, blocks []diagramBlock, rendered map[int]string) string {
	return reassembleDiagram(lines, blocks, rendered, "mermaid diagram")
}

// preprocessMermaid scans raw markdown for ```mermaid blocks and replaces them
// with ![mermaid diagram](path.png) using the given renderer.
// Blocks are rendered in parallel. On render error, the original code block is preserved.
func preprocessMermaid(markdown string, renderer *MermaidRenderer) string {
	if renderer == nil {
		return markdown
	}

	lines, blocks := extractDiagramBlocks(markdown, mermaidFenceRe)
	if len(blocks) == 0 {
		return markdown
	}

	rendered := renderDiagramBlocks(blocks, renderer)
	return reassembleDiagram(lines, blocks, rendered, "mermaid diagram")
}
