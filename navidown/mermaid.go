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
	opts     MermaidOptions
	cache    sync.Map // sha256 hex -> absolute PNG path
	cacheDir string
	mmdcPath string
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

	cacheDir, err := os.MkdirTemp("", "navidown-mermaid-")
	if err != nil {
		return nil
	}

	return &MermaidRenderer{
		opts:     opts,
		cacheDir: cacheDir,
		mmdcPath: mmdcPath,
	}
}

// RenderToFile renders mermaid source to a PNG file and returns its absolute path.
// Results are cached by content hash.
func (r *MermaidRenderer) RenderToFile(source string) (string, error) {
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(source)))

	if cached, ok := r.cache.Load(hash); ok {
		path, _ := cached.(string)
		return path, nil
	}

	inputPath := filepath.Join(r.cacheDir, hash+".mmd")
	outputPath := filepath.Join(r.cacheDir, hash+".png")

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

	r.cache.Store(hash, outputPath)
	return outputPath, nil
}

// Close removes the temporary cache directory and all rendered PNGs.
func (r *MermaidRenderer) Close() {
	if r.cacheDir != "" {
		_ = os.RemoveAll(r.cacheDir)
	}
}

// mermaidFenceRe matches the opening of a mermaid fenced code block.
var mermaidFenceRe = regexp.MustCompile("^(\\s*`{3,})mermaid\\s*$")

// preprocessMermaid scans raw markdown for ```mermaid blocks and replaces them
// with ![mermaid diagram](path.png) using the given renderer.
// On render error, the original code block is preserved.
func preprocessMermaid(markdown string, renderer *MermaidRenderer) string {
	if renderer == nil {
		return markdown
	}

	lines := strings.Split(markdown, "\n")
	var result strings.Builder
	result.Grow(len(markdown))

	i := 0
	for i < len(lines) {
		match := mermaidFenceRe.FindStringSubmatch(lines[i])
		if match == nil {
			result.WriteString(lines[i])
			if i < len(lines)-1 {
				result.WriteByte('\n')
			}
			i++
			continue
		}

		fencePrefix := match[1] // the backtick sequence (with optional leading whitespace)
		// extract just the backticks to determine fence length
		backticks := strings.TrimLeft(fencePrefix, " \t")
		fenceLen := len(backticks)
		openLine := i
		i++

		// collect mermaid source lines until closing fence
		var source strings.Builder
		closed := false
		for i < len(lines) {
			// closing fence: same or more backticks, no info string
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

		if !closed {
			// unclosed fence — write original lines as-is
			for j := openLine; j < i; j++ {
				result.WriteString(lines[j])
				if j < len(lines)-1 {
					result.WriteByte('\n')
				}
			}
			continue
		}

		// render mermaid to PNG
		pngPath, err := renderer.RenderToFile(source.String())
		if err != nil {
			// graceful degradation: keep original code block
			for j := openLine; j < i; j++ {
				result.WriteString(lines[j])
				if j < len(lines)-1 {
					result.WriteByte('\n')
				}
			}
			continue
		}

		result.WriteString("![mermaid diagram](" + pngPath + ")")
		if i < len(lines) {
			result.WriteByte('\n')
		}
	}

	return result.String()
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
