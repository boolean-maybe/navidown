package navidown

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
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

//go:embed mermaid_state.css
var stateMermaidCSS []byte

//go:embed mermaid_large.css
var largeMermaidCSS []byte

//go:embed mermaid_gantt.css
var ganttMermaidCSS []byte

//go:embed mermaid_pie.css
var pieMermaidCSS []byte

//go:embed mermaid_git.css
var gitMermaidCSS []byte

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

// stateConfigData returns config JSON with fontSize for state diagrams.
// Uses 36px so mermaid sizes boxes generously; CSS overrides visible text to 20px.
func (o *MermaidOptions) stateConfigData() []byte {
	return configDataWithFontSize(o.resolvedTheme(), "36px")
}

// ganttConfigData returns config JSON with gantt-specific settings.
func (o *MermaidOptions) ganttConfigData() []byte {
	var cfg map[string]any
	_ = json.Unmarshal(defaultMermaidConfig, &cfg)
	cfg["theme"] = o.resolvedTheme()
	cfg["gantt"] = map[string]any{
		"fontSize":             8,
		"sectionFontSize":      6,
		"barHeight":            18,
		"barGap":               5,
		"topPadding":           40,
		"bottomPadding":        60,
		"gridLineStartPadding": 20,
	}
	data, _ := json.MarshalIndent(cfg, "", "  ")
	return data
}

// largeConfigData returns config JSON with fontSize for ER diagrams.
func (o *MermaidOptions) largeConfigData() []byte {
	return configDataWithFontSize(o.resolvedTheme(), "18px")
}

// pieConfigData returns config JSON with colorful pie slice colors.
func (o *MermaidOptions) pieConfigData() []byte {
	var cfg map[string]any
	_ = json.Unmarshal(defaultMermaidConfig, &cfg)
	cfg["theme"] = o.resolvedTheme()
	if tv, ok := cfg["themeVariables"].(map[string]any); ok {
		tv["pie1"] = "#89b4fa"
		tv["pie2"] = "#f38ba8"
		tv["pie3"] = "#a6e3a1"
		tv["pie4"] = "#f9e2af"
		tv["pie5"] = "#cba6f7"
		tv["pie6"] = "#94e2d5"
		tv["pie7"] = "#fab387"
		tv["pie8"] = "#74c7ec"
		tv["pie9"] = "#f2cdcd"
		tv["pie10"] = "#b4befe"
		tv["pie11"] = "#eba0ac"
		tv["pie12"] = "#89dceb"
		tv["pieStrokeColor"] = "#ffffff"
		tv["pieStrokeWidth"] = "2px"
		tv["pieOuterStrokeColor"] = "#ffffff"
		tv["pieOuterStrokeWidth"] = "2px"
		tv["pieOpacity"] = "0.9"
	}
	data, _ := json.MarshalIndent(cfg, "", "  ")
	return data
}

// gitConfigData returns config JSON with colorful branch colors and dark-mode
// branch labels. SVG attribute overrides (circle r, stroke-width) are applied
// via post-processing since mmdc's Puppeteer ignores CSS for SVG attributes.
func (o *MermaidOptions) gitConfigData() []byte {
	var cfg map[string]any
	_ = json.Unmarshal(defaultMermaidConfig, &cfg)
	cfg["theme"] = o.resolvedTheme()
	if tv, ok := cfg["themeVariables"].(map[string]any); ok {
		tv["git0"] = "#89b4fa"
		tv["git1"] = "#f38ba8"
		tv["git2"] = "#a6e3a1"
		tv["git3"] = "#f9e2af"
		tv["git4"] = "#cba6f7"
		tv["git5"] = "#94e2d5"
		tv["git6"] = "#fab387"
		tv["git7"] = "#74c7ec"
		for i := 0; i <= 7; i++ {
			tv[fmt.Sprintf("gitBranchLabel%d", i)] = "#ffffff"
		}
		tv["commitLabelColor"] = "#ffffff"
	}
	data, _ := json.MarshalIndent(cfg, "", "  ")
	return data
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
	stateConfigPath string // path to state-diagram config JSON
	stateConfigData []byte // resolved state config content
	stateCSSPath    string // path to state-diagram CSS override
	largeConfigPath string // path to large-font config JSON (ER)
	largeConfigData []byte // resolved large config content
	largeCSSPath    string // path to large-font CSS override (ER)
	ganttConfigPath string // path to gantt config JSON
	ganttConfigData []byte // resolved gantt config content
	ganttCSSPath    string // path to gantt CSS override
	pieConfigPath   string // path to pie config JSON
	pieConfigData   []byte // resolved pie config content
	pieCSSPath      string // path to pie CSS override
	gitConfigPath   string // path to git config JSON
	gitConfigData   []byte // resolved git config content
	gitCSSPath      string // path to git CSS override
	resvgPath       string // path to resvg binary; "" if not found
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

	stateConfigData := opts.stateConfigData()
	stateConfigPath := filepath.Join(workDir, "mermaid-state-config.json")
	if err := os.WriteFile(stateConfigPath, stateConfigData, 0600); err != nil {
		return nil
	}

	stateCSSPath := filepath.Join(workDir, "mermaid-state.css")
	if err := os.WriteFile(stateCSSPath, stateMermaidCSS, 0600); err != nil {
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

	ganttConfigData := opts.ganttConfigData()
	ganttConfigPath := filepath.Join(workDir, "mermaid-gantt-config.json")
	if err := os.WriteFile(ganttConfigPath, ganttConfigData, 0600); err != nil {
		return nil
	}

	ganttCSSPath := filepath.Join(workDir, "mermaid-gantt.css")
	if err := os.WriteFile(ganttCSSPath, ganttMermaidCSS, 0600); err != nil {
		return nil
	}

	pieConfigData := opts.pieConfigData()
	pieConfigPath := filepath.Join(workDir, "mermaid-pie-config.json")
	if err := os.WriteFile(pieConfigPath, pieConfigData, 0600); err != nil {
		return nil
	}

	pieCSSPath := filepath.Join(workDir, "mermaid-pie.css")
	if err := os.WriteFile(pieCSSPath, pieMermaidCSS, 0600); err != nil {
		return nil
	}

	gitConfigData := opts.gitConfigData()
	gitConfigPath := filepath.Join(workDir, "mermaid-git-config.json")
	if err := os.WriteFile(gitConfigPath, gitConfigData, 0600); err != nil {
		return nil
	}

	gitCSSPath := filepath.Join(workDir, "mermaid-git.css")
	if err := os.WriteFile(gitCSSPath, gitMermaidCSS, 0600); err != nil {
		return nil
	}

	resvgPath, _ := exec.LookPath("resvg")

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
		stateConfigPath: stateConfigPath,
		stateConfigData: stateConfigData,
		stateCSSPath:    stateCSSPath,
		largeConfigPath: largeConfigPath,
		largeConfigData: largeConfigData,
		largeCSSPath:    largeCSSPath,
		ganttConfigPath: ganttConfigPath,
		ganttConfigData: ganttConfigData,
		ganttCSSPath:    ganttCSSPath,
		pieConfigPath:   pieConfigPath,
		pieConfigData:   pieConfigData,
		pieCSSPath:      pieCSSPath,
		gitConfigPath:   gitConfigPath,
		gitConfigData:   gitConfigData,
		gitCSSPath:      gitCSSPath,
		resvgPath:       resvgPath,
	}
}

// diagramFontTier classifies diagram source into font size tiers.
type diagramFontTier int

const (
	tierDefault diagramFontTier = iota // flowchart, sequence — compact 8px
	tierClass                          // class diagram — medium 10px
	tierState                          // state diagram — 36px config for box sizing, 20px visible text
	tierLarge                          // ER diagram — large 18px
	tierGantt                          // gantt — compact bars with smaller task/section fonts
	tierPie                            // pie — colorful slices with white borders
	tierGit                            // gitGraph — SVG post-processed for thin lines/circles
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
		if strings.HasPrefix(trimmed, "stateDiagram") {
			return tierState
		}
		if strings.HasPrefix(trimmed, "erDiagram") {
			return tierLarge
		}
		if strings.HasPrefix(trimmed, "gantt") {
			return tierGantt
		}
		if strings.HasPrefix(trimmed, "pie") {
			return tierPie
		}
		if strings.HasPrefix(trimmed, "gitGraph") {
			return tierGit
		}
		return tierDefault
	}
	return tierDefault
}

// widthForSource returns a wider page width for diagram types that benefit
// from horizontal space (e.g. gantt charts). Returns the default for others.
func (r *MermaidRenderer) widthForSource(source string) int {
	if isWideLayout(source) {
		w := r.opts.resolvedWidth()
		if w < 1000 {
			return 1000
		}
		return w
	}
	return r.opts.resolvedWidth()
}

// isWideLayout returns true for diagram types that need extra horizontal space.
func isWideLayout(source string) bool {
	for _, line := range strings.SplitN(source, "\n", 10) {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "%%") {
			continue
		}
		return strings.HasPrefix(trimmed, "gantt")
	}
	return false
}

// configForSource returns the config and CSS file paths appropriate for the
// diagram type in source.
func (r *MermaidRenderer) configForSource(source string) (configPath, cssPath string) {
	switch fontTier(source) {
	case tierClass:
		return r.classConfigPath, r.classCSSPath
	case tierState:
		return r.stateConfigPath, r.stateCSSPath
	case tierLarge:
		return r.largeConfigPath, r.largeCSSPath
	case tierGantt:
		return r.ganttConfigPath, r.ganttCSSPath
	case tierPie:
		return r.pieConfigPath, r.pieCSSPath
	case tierGit:
		return r.gitConfigPath, r.gitCSSPath
	default:
		return r.configPath, r.cssPath
	}
}

// cacheKey computes a hash incorporating the mermaid source and render options
// (theme, background, scale) so that option changes don't produce stale hits.
// cacheVersion must be bumped when post-processing logic changes, since
// post-processing runs after the cache key is computed from inputs.
const cacheVersion = "v5"

func (r *MermaidRenderer) cacheKey(source string) string {
	h := sha256.New()
	h.Write([]byte(cacheVersion))
	h.Write([]byte{0})
	h.Write([]byte(source))
	h.Write([]byte{0}) // separator
	switch fontTier(source) {
	case tierClass:
		h.Write(r.classConfigData)
		h.Write([]byte{0})
		h.Write(classMermaidCSS)
	case tierState:
		h.Write(r.stateConfigData)
		h.Write([]byte{0})
		h.Write(stateMermaidCSS)
	case tierLarge:
		h.Write(r.largeConfigData)
		h.Write([]byte{0})
		h.Write(largeMermaidCSS)
	case tierGantt:
		h.Write(r.ganttConfigData)
		h.Write([]byte{0})
		h.Write(ganttMermaidCSS)
	case tierPie:
		h.Write(r.pieConfigData)
		h.Write([]byte{0})
		h.Write(pieMermaidCSS)
	case tierGit:
		h.Write(r.gitConfigData)
		h.Write([]byte{0})
		h.Write(gitMermaidCSS)
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
	_, _ = fmt.Fprintf(h, "%d", r.widthForSource(source))
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

	// gitGraph requires SVG post-processing + resvg because mmdc's Puppeteer
	// ignores CSS overrides for SVG attributes (circle r, stroke-width).
	if fontTier(source) == tierGit && r.resvgPath != "" {
		pngPath, err := r.renderViaResvg(ctx, inputPath, outputPath, cfgPath, css, source)
		if err != nil {
			return "", err
		}
		_ = os.Remove(inputPath)
		r.cache.Store(key, pngPath)
		return pngPath, nil
	}

	cmd := exec.CommandContext(ctx, r.mmdcPath, // #nosec G204 -- mmdcPath from LookPath("mmdc") or user-provided
		"-i", inputPath,
		"-o", outputPath,
		"-c", cfgPath,
		"-C", css,
		"-b", r.opts.resolvedBackground(),
		"-w", fmt.Sprintf("%d", r.widthForSource(source)),
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

// renderViaResvg renders mermaid source to SVG via mmdc, post-processes the SVG
// to fix attributes that CSS cannot override (circle radii, stroke widths),
// then rasterizes to PNG via resvg.
func (r *MermaidRenderer) renderViaResvg(ctx context.Context, inputPath, outputPath, cfgPath, css, source string) (string, error) {
	svgPath := strings.TrimSuffix(outputPath, ".png") + ".svg"

	cmd := exec.CommandContext(ctx, r.mmdcPath, // #nosec G204
		"-i", inputPath,
		"-o", svgPath,
		"-c", cfgPath,
		"-C", css,
		"-b", r.opts.resolvedBackground(),
		"-w", fmt.Sprintf("%d", r.widthForSource(source)),
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("mmdc (svg) failed: %w\n%s", err, string(out))
	}

	svgData, err := os.ReadFile(svgPath)
	if err != nil {
		return "", fmt.Errorf("read svg: %w", err)
	}
	_ = os.Remove(svgPath)

	svgData = postProcessGitSVG(svgData)

	// rasterize with resvg at 1.5x for a compact but readable result
	_, vbW, ok := extractSVGViewBoxWidth(svgData)
	targetWidth := r.widthForSource(source) * 3 / 2
	if ok && vbW > 0 {
		targetWidth = int(vbW) * 3 / 2
	}

	resvgCmd := exec.CommandContext(ctx, r.resvgPath, // #nosec G204
		"-w", strconv.Itoa(targetWidth),
		"-", outputPath,
	)
	resvgCmd.Stdin = bytes.NewReader(svgData)

	resvgOut, err := resvgCmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("resvg failed: %w\n%s", err, string(resvgOut))
	}

	return outputPath, nil
}

// postProcessGitSVG modifies SVG attributes that CSS cannot override in
// mmdc's Puppeteer rendering: circle radii, arrow/branch stroke widths,
// and branch label box sizing.
func postProcessGitSVG(svg []byte) []byte {
	s := string(svg)
	// shrink commit circles: r="10" → r="4", r="9" → r="4", r="6" → r="2"
	s = strings.ReplaceAll(s, `r="10"`, `r="4"`)
	s = strings.ReplaceAll(s, `r="9"`, `r="4"`)
	s = strings.ReplaceAll(s, `r="6"`, `r="2"`)
	// thin arrow lines: stroke-width:8 → stroke-width:1
	s = strings.ReplaceAll(s, "stroke-width:8", "stroke-width:1")
	// thin commit-reverse stroke: stroke-width:3 → stroke-width:1
	s = strings.ReplaceAll(s, "stroke-width:3;", "stroke-width:1;")
	// hide label boxes, force monospace font, brighten text
	s = patchGitStyles(s)
	return []byte(s)
}

// patchGitStyles applies direct SVG element and inline CSS edits to force
// monospace font, bright white text, and hidden label boxes. resvg has
// limited CSS support (no !important, no #id selectors, no var()), so we
// must patch the <style> rules and inject inline style attributes.
func patchGitStyles(s string) string {
	// --- patch inline <style> block ---
	// hide commit-label background
	s = strings.Replace(s, "fill:#374151;opacity:0.5;", "fill:none;opacity:0;", 1)
	// force monospace in all CSS font-family declarations
	s = strings.ReplaceAll(s, `font-family:'trebuchet ms',verdana,arial,sans-serif`, `font-family:monospace`)
	s = strings.ReplaceAll(s, `font-family:"trebuchet ms",verdana,arial,sans-serif`, `font-family:monospace`)
	s = strings.ReplaceAll(s, `font-family:var(--mermaid-font-family)`, `font-family:monospace`)
	// brighten all CSS fill values to pure white
	s = strings.ReplaceAll(s, "fill:lightgrey", "fill:#ffffff")
	s = strings.ReplaceAll(s, "fill:#cdd6f4", "fill:#ffffff")
	s = strings.ReplaceAll(s, "fill:#f9fafb", "fill:#ffffff")

	// --- inject inline style on elements (resvg respects these over CSS) ---
	const textStyle = `style="font-family:monospace;fill:#ffffff" `
	// commit labels: <text ... class="commit-label">
	s = strings.ReplaceAll(s, `class="commit-label">`, textStyle+`class="commit-label">`)
	// branch label tspans inherit font from parent <text>
	s = strings.ReplaceAll(s, `<text><tspan`, `<text `+textStyle+`><tspan`)

	// hide branchLabelBkg rects with inline style
	s = strings.ReplaceAll(s, `class="branchLabelBkg `, `style="fill:none;stroke:none" class="branchLabelBkg `)
	return s
}

// extractSVGViewBoxWidth extracts the viewBox width from SVG data.
func extractSVGViewBoxWidth(svg []byte) (string, float64, bool) {
	// quick regex to find viewBox="minX minY width height"
	re := regexp.MustCompile(`viewBox="[^"]*\s+([\d.]+)\s+[\d.]+"`)
	m := re.FindSubmatch(svg)
	if m == nil {
		return "", 0, false
	}
	w, err := strconv.ParseFloat(string(m[1]), 64)
	if err != nil {
		return "", 0, false
	}
	return string(m[1]), w, true
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
