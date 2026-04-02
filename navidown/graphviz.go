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

// GraphvizOptions configures graphviz diagram rendering.
type GraphvizOptions struct {
	DotPath  string        // path to dot binary; "" = lookup "dot" in PATH
	Layout   string        // layout engine; "" = "dot"
	DPI      int           // render DPI; 0 = 144 (retina)
	Timeout  time.Duration // render timeout; 0 = 30s
	CacheDir string        // persistent cache dir; "" = auto (os.UserCacheDir()/navidown/graphviz)
	// DarkMode applies dark-friendly defaults: transparent background, white
	// text/edges, dark node fill. Set to false for light terminals or when the
	// dot source defines its own colors. Default true.
	DarkMode *bool
}

func (o *GraphvizOptions) resolvedDarkMode() bool {
	if o.DarkMode != nil {
		return *o.DarkMode
	}
	return true
}

func (o *GraphvizOptions) resolvedLayout() string {
	if o.Layout != "" {
		return o.Layout
	}
	return "dot"
}

func (o *GraphvizOptions) resolvedDPI() int {
	if o.DPI > 0 {
		return o.DPI
	}
	return 144
}

func (o *GraphvizOptions) resolvedTimeout() time.Duration {
	if o.Timeout > 0 {
		return o.Timeout
	}
	return 30 * time.Second
}

// GraphvizRenderer renders graphviz dot source code to PNG files.
type GraphvizRenderer struct {
	opts          GraphvizOptions
	cache         sync.Map
	persistentDir string
	tempDir       string
	workDir       string
	dotPath       string
}

// NewGraphvizRenderer creates a new renderer. Returns nil if dot is not found.
func NewGraphvizRenderer(opts GraphvizOptions) *GraphvizRenderer {
	dotPath := opts.DotPath
	if dotPath == "" {
		resolved, err := exec.LookPath("dot")
		if err != nil {
			return nil
		}
		dotPath = resolved
	}

	persistentDir, tempDir, workDir := resolveCacheDir(opts.CacheDir, "graphviz")
	if workDir == "" {
		return nil
	}

	return &GraphvizRenderer{
		opts:          opts,
		persistentDir: persistentDir,
		tempDir:       tempDir,
		workDir:       workDir,
		dotPath:       dotPath,
	}
}

func (r *GraphvizRenderer) cacheKey(source string) string {
	h := sha256.New()
	h.Write([]byte(source))
	h.Write([]byte{0})
	h.Write([]byte(r.opts.resolvedLayout()))
	h.Write([]byte{0})
	_, _ = fmt.Fprintf(h, "%d", r.opts.resolvedDPI())
	h.Write([]byte{0})
	_, _ = fmt.Fprintf(h, "%t", r.opts.resolvedDarkMode())
	return fmt.Sprintf("%x", h.Sum(nil))
}

// RenderToFile renders dot source to a PNG file and returns its absolute path.
// Results are cached by content hash (in-memory and on disk).
func (r *GraphvizRenderer) RenderToFile(source string) (string, error) {
	key := r.cacheKey(source)
	outputPath := filepath.Join(r.workDir, key+".png")

	if cached, ok := r.cache.Load(key); ok {
		if path, ok := cached.(string); ok {
			return path, nil
		}
	}

	if _, err := os.Stat(outputPath); err == nil {
		r.cache.Store(key, outputPath)
		return outputPath, nil
	}

	inputPath := filepath.Join(r.workDir, key+".dot")
	if err := os.WriteFile(inputPath, []byte(source), 0600); err != nil {
		return "", fmt.Errorf("write dot source: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), r.opts.resolvedTimeout())
	defer cancel()

	args := []string{
		"-Tpng",
		"-Gbgcolor=transparent",
		fmt.Sprintf("-Gdpi=%d", r.opts.resolvedDPI()),
		fmt.Sprintf("-K%s", r.opts.resolvedLayout()),
	}
	if r.opts.resolvedDarkMode() {
		args = append(args,
			"-Gcolor=white",
			"-Ncolor=white", "-Nfontcolor=white", "-Nstyle=filled", "-Nfillcolor=#333333",
			"-Ecolor=white", "-Efontcolor=white",
		)
	}
	args = append(args, "-o", outputPath, inputPath)

	cmd := exec.CommandContext(ctx, r.dotPath, args...) // #nosec G204 -- dotPath from LookPath("dot") or user-provided

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("dot failed: %w\n%s", err, string(out))
	}

	_ = os.Remove(inputPath)

	r.cache.Store(key, outputPath)
	return outputPath, nil
}

// ClearCache flushes the in-memory cache and removes disk-cached PNGs.
func (r *GraphvizRenderer) ClearCache() {
	r.cache.Range(func(key, _ any) bool { r.cache.Delete(key); return true })
	entries, _ := os.ReadDir(r.workDir)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".png") {
			_ = os.Remove(filepath.Join(r.workDir, e.Name()))
		}
	}
}

// WorkDir returns the cache working directory.
func (r *GraphvizRenderer) WorkDir() string { return r.workDir }

// EvictKeys removes specific cache entries by key (hex hash, no extension).
func (r *GraphvizRenderer) EvictKeys(keys []string) {
	for _, key := range keys {
		r.cache.Delete(key)
		_ = os.Remove(filepath.Join(r.workDir, key+".png"))
	}
}

// Close releases resources. Only removes the temp dir (if used as fallback).
func (r *GraphvizRenderer) Close() {
	if r.tempDir != "" {
		_ = os.RemoveAll(r.tempDir)
	}
}

// graphvizFenceRe matches the opening of a dot/graphviz fenced code block.
var graphvizFenceRe = regexp.MustCompile(`^(\s*` + "`{3,}" + `)(?:dot|graphviz)\s*$`)

// preprocessGraphviz scans raw markdown for ```dot / ```graphviz blocks and
// replaces them with ![dot diagram](path.png) using the given renderer.
func preprocessGraphviz(markdown string, renderer *GraphvizRenderer) string {
	if renderer == nil {
		return markdown
	}

	lines, blocks := extractDiagramBlocks(markdown, graphvizFenceRe)
	if len(blocks) == 0 {
		return markdown
	}

	rendered := renderDiagramBlocks(blocks, renderer)
	return reassembleDiagram(lines, blocks, rendered, "dot diagram")
}
