package navidown

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

//go:embed mermaid_config.json
var defaultMermaidConfig []byte

//go:embed mermaid_override.css
var defaultMermaidCSS []byte

//go:embed mermaid_class.css
var classMermaidCSS []byte

//go:embed mermaid_large.css
var largeMermaidCSS []byte

// MermaidOptions configures mermaid diagram rendering.
type MermaidOptions struct {
	MmdcPath        string        // path to mmdc binary; "" = lookup "mmdc" in PATH
	Theme           string        // mermaid theme; "" = "base" (uses embedded config themeVariables)
	BackgroundColor string        // background color; "" = "transparent"
	Scale           int           // render scale; 0 = 2 (retina)
	Width           int           // page width in CSS pixels; 0 = 600
	Timeout         time.Duration // render timeout; 0 = 30s
	CacheDir        string        // persistent cache dir; "" = auto (os.UserCacheDir()/navidown/mermaid)
}

func (o *MermaidOptions) resolvedTheme() string {
	if o.Theme != "" {
		return o.Theme
	}
	return "base"
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

func (o *MermaidOptions) resolvedWidth() int {
	if o.Width > 0 {
		return o.Width
	}
	return 450
}

func (o *MermaidOptions) resolvedTimeout() time.Duration {
	if o.Timeout > 0 {
		return o.Timeout
	}
	return 30 * time.Second
}

// resolvedConfigData returns the embedded default config JSON with the theme
// field set to resolvedTheme(). The result is deterministic for a given theme.
func (o *MermaidOptions) resolvedConfigData() []byte {
	return configDataWithFontSize(o.resolvedTheme(), "")
}

// classConfigData returns config JSON with fontSize for class diagrams.
func (o *MermaidOptions) classConfigData() []byte {
	return configDataWithFontSize(o.resolvedTheme(), "10px")
}

// largeConfigData returns config JSON with fontSize for state/ER diagrams.
func (o *MermaidOptions) largeConfigData() []byte {
	return configDataWithFontSize(o.resolvedTheme(), "18px")
}

// configDataWithFontSize returns the embedded config with the given theme and
// optional fontSize override applied to themeVariables.
func configDataWithFontSize(theme, fontSize string) []byte {
	var cfg map[string]any
	_ = json.Unmarshal(defaultMermaidConfig, &cfg)
	cfg["theme"] = theme
	if fontSize != "" {
		if tv, ok := cfg["themeVariables"].(map[string]any); ok {
			tv["fontSize"] = fontSize
		}
	}
	data, _ := json.MarshalIndent(cfg, "", "  ")
	return data
}

// MermaidRenderer renders mermaid source code to PNG files using mmdc.
type MermaidRenderer struct {
	opts            MermaidOptions
	cache           sync.Map // cache key hex -> absolute PNG path
	persistentDir   string   // persistent cache (never deleted by Close)
	tempDir         string   // temp fallback (deleted by Close); "" if persistent worked
	workDir         string   // whichever dir is actually used
	mmdcPath        string
	configPath      string // path to config JSON written to workDir
	configData      []byte // resolved config content (used in cache key)
	classConfigPath string // path to class-diagram config JSON
	classConfigData []byte // resolved class config content (used in cache key)
	cssPath         string // path to default CSS override written to workDir
	classCSSPath    string // path to class-diagram-specific CSS override
	largeConfigPath string // path to large-font config JSON (state/ER)
	largeConfigData []byte // resolved large config content
	largeCSSPath    string // path to large-font CSS override (state/ER)
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

	configData := opts.resolvedConfigData()
	configPath := filepath.Join(workDir, "mermaid-config.json")
	if err := os.WriteFile(configPath, configData, 0600); err != nil {
		return nil
	}

	classConfigData := opts.classConfigData()
	classConfigPath := filepath.Join(workDir, "mermaid-class-config.json")
	if err := os.WriteFile(classConfigPath, classConfigData, 0600); err != nil {
		return nil
	}

	cssPath := filepath.Join(workDir, "mermaid-override.css")
	if err := os.WriteFile(cssPath, defaultMermaidCSS, 0600); err != nil {
		return nil
	}

	classCSSPath := filepath.Join(workDir, "mermaid-class.css")
	if err := os.WriteFile(classCSSPath, classMermaidCSS, 0600); err != nil {
		return nil
	}

	largeConfigData := opts.largeConfigData()
	largeConfigPath := filepath.Join(workDir, "mermaid-large-config.json")
	if err := os.WriteFile(largeConfigPath, largeConfigData, 0600); err != nil {
		return nil
	}

	largeCSSPath := filepath.Join(workDir, "mermaid-large.css")
	if err := os.WriteFile(largeCSSPath, largeMermaidCSS, 0600); err != nil {
		return nil
	}

	return &MermaidRenderer{
		opts:            opts,
		persistentDir:   persistentDir,
		tempDir:         tempDir,
		workDir:         workDir,
		mmdcPath:        mmdcPath,
		configPath:      configPath,
		configData:      configData,
		classConfigPath: classConfigPath,
		classConfigData: classConfigData,
		cssPath:         cssPath,
		classCSSPath:    classCSSPath,
		largeConfigPath: largeConfigPath,
		largeConfigData: largeConfigData,
		largeCSSPath:    largeCSSPath,
	}
}

// diagramFontTier classifies diagram source into font size tiers.
type diagramFontTier int

const (
	tierDefault diagramFontTier = iota // flowchart, sequence — compact 8px
	tierClass                          // class diagram — medium 10px
	tierLarge                          // state, ER — large 14px
)

// fontTier returns the font size tier for the given mermaid source.
func fontTier(source string) diagramFontTier {
	for _, line := range strings.SplitN(source, "\n", 10) {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "%%") {
			continue
		}
		if strings.HasPrefix(trimmed, "classDiagram") {
			return tierClass
		}
		for _, prefix := range []string{"stateDiagram", "erDiagram"} {
			if strings.HasPrefix(trimmed, prefix) {
				return tierLarge
			}
		}
		return tierDefault
	}
	return tierDefault
}

// configForSource returns the config and CSS file paths appropriate for the
// diagram type in source.
func (r *MermaidRenderer) configForSource(source string) (configPath, cssPath string) {
	switch fontTier(source) {
	case tierClass:
		return r.classConfigPath, r.classCSSPath
	case tierLarge:
		return r.largeConfigPath, r.largeCSSPath
	default:
		return r.configPath, r.cssPath
	}
}

// cacheKey computes a hash incorporating the mermaid source and render options
// (theme, background, scale) so that option changes don't produce stale hits.
func (r *MermaidRenderer) cacheKey(source string) string {
	h := sha256.New()
	h.Write([]byte(source))
	h.Write([]byte{0}) // separator
	switch fontTier(source) {
	case tierClass:
		h.Write(r.classConfigData)
		h.Write([]byte{0})
		h.Write(classMermaidCSS)
	case tierLarge:
		h.Write(r.largeConfigData)
		h.Write([]byte{0})
		h.Write(largeMermaidCSS)
	default:
		h.Write(r.configData)
		h.Write([]byte{0})
		h.Write(defaultMermaidCSS)
	}
	h.Write([]byte{0})
	h.Write([]byte(r.opts.resolvedBackground()))
	h.Write([]byte{0})
	_, _ = fmt.Fprintf(h, "%d", r.opts.resolvedScale())
	h.Write([]byte{0})
	_, _ = fmt.Fprintf(h, "%d", r.opts.resolvedWidth())
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

	cfgPath, css := r.configForSource(source)
	cmd := exec.CommandContext(ctx, r.mmdcPath, // #nosec G204 -- mmdcPath from LookPath("mmdc") or user-provided
		"-i", inputPath,
		"-o", outputPath,
		"-c", cfgPath,
		"-C", css,
		"-b", r.opts.resolvedBackground(),
		"-w", fmt.Sprintf("%d", r.opts.resolvedWidth()),
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
