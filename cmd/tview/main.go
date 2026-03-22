package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/boolean-maybe/navidown/loaders"
	"github.com/boolean-maybe/navidown/navidown"
	tviewAdapter "github.com/boolean-maybe/navidown/navidown/tview"
	"github.com/boolean-maybe/navidown/util"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func main() {
	// parse flags
	syntaxTheme := flag.String("syntax-theme", "", "chroma style name for code block syntax highlighting (e.g. dracula, monokai, catppuccin-macchiato)")
	syntaxBg := flag.String("syntax-background", "", "background color for code blocks (e.g. #282a36, 236)")
	syntaxBorder := flag.String("syntax-border", "", "border color for code blocks (e.g. #6272a4, 244)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [flags] <file-path-or-url>\n\nflags:\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	arg := flag.Arg(0)

	// load initial content
	content, sourcePath, err := loadContent(arg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading content: %v\n", err)
		os.Exit(1)
	}

	// create tview application
	app := tview.NewApplication()

	// create markdown viewer
	mdViewer := tviewAdapter.NewTextView()

	// set custom ANSI converter for proper background color support
	mdViewer.SetAnsiConverter(util.NewAnsiConverter(true))

	// enable Kitty image protocol support
	imgResolver := navidown.NewImageResolver([]string{"."})
	imgManager := tviewAdapter.NewImageManager(imgResolver, 8, 16)
	// Allow images to take up to their natural size (0 = no limit)
	imgManager.SetMaxRows(40)
	imgManager.SetSupported(true)
	mdViewer.SetImageManager(imgManager)

	// apply syntax highlighting overrides if specified
	if *syntaxTheme != "" || *syntaxBg != "" || *syntaxBorder != "" {
		renderer := navidown.NewANSIRenderer()
		if *syntaxTheme != "" {
			renderer = renderer.WithCodeTheme(*syntaxTheme)
		}
		if *syntaxBg != "" {
			renderer = renderer.WithCodeBackground(*syntaxBg)
		}
		if *syntaxBorder != "" {
			renderer = renderer.WithCodeBorder(*syntaxBorder)
		}
		mdViewer.Core().SetRenderer(renderer)
	}

	// enable mermaid diagram rendering (requires mmdc in PATH)
	mdViewer.Core().SetMermaidOptions(&navidown.MermaidOptions{Theme: "dark"})

	// enable graphviz diagram rendering (requires dot in PATH)
	mdViewer.Core().SetGraphvizOptions(&navidown.GraphvizOptions{})

	// set up content fetcher for link navigation
	provider := &loaders.FileHTTP{SearchRoots: []string{"."}}

	// wire up link activation handler - manually fetch and update through adapter
	mdViewer.SetSelectHandler(func(v *tviewAdapter.TextViewViewer, elem navidown.NavElement) {
		if elem.Type != navidown.NavElementURL {
			return
		}

		// handle internal anchor links (same file)
		if elem.IsInternalLink() {
			v.ScrollToAnchor(elem.AnchorTarget(), true)
			return
		}

		// parse fragment from URL
		path, fragment := splitFragment(elem.URL)

		// create elem for fetching (without fragment)
		elemForFetch := elem
		elemForFetch.URL = path

		// fetch content for external links
		content, err := provider.FetchContent(elemForFetch)
		if err != nil {
			errorContent := "# Error\n\nFailed to load `" + elem.URL + "`:\n\n```\n" + err.Error() + "\n```"
			v.SetMarkdownWithSource(errorContent, elem.SourceFilePath, true)
			return
		}

		if content == "" {
			return
		}

		// resolve path (use path without fragment)
		newSourcePath := path
		if !strings.HasPrefix(path, "http://") && !strings.HasPrefix(path, "https://") && elem.SourceFilePath != "" {
			resolved, rerr := navidown.ResolveMarkdownPath(path, elem.SourceFilePath, []string{"."})
			if rerr == nil && resolved != "" {
				newSourcePath = resolved
			}
		}

		// update through adapter (this will refresh display)
		v.SetMarkdownWithSource(content, newSourcePath, true)

		// scroll to anchor after load
		if fragment != "" {
			v.ScrollToAnchor(fragment, false)
		}
	})

	// create status bar
	statusBar := tview.NewTextView()
	statusBar.SetDynamicColors(true)
	statusBar.SetTextAlign(tview.AlignLeft)

	mdViewer.SetStateChangedHandler(func(v *tviewAdapter.TextViewViewer) {
		updateStatusBar(statusBar, v)
	})

	// load initial content
	mdViewer.SetMarkdownWithSource(content, sourcePath, false)

	// initial status bar update
	updateStatusBar(statusBar, mdViewer)

	// create flex layout with status bar
	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(mdViewer, 0, 1, true).
		AddItem(statusBar, 1, 0, false)

	// set up quit handler
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 'q' {
			app.Stop()
			return nil
		}
		return event
	})

	// run application
	if err := app.SetRoot(flex, true).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error running application: %v\n", err)
		os.Exit(1)
	}
}

// loadContent loads content from a file path or URL.
func loadContent(arg string) (content string, sourcePath string, err error) {
	// check if it's an HTTP(S) URL
	if strings.HasPrefix(arg, "http://") || strings.HasPrefix(arg, "https://") {
		provider := &loaders.FileHTTP{}
		content, err := provider.FetchContent(navidown.NavElement{URL: arg})
		return content, arg, err
	}

	// local file
	absPath, err := filepath.Abs(arg)
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve path: %w", err)
	}

	data, err := os.ReadFile(absPath) // #nosec G703 -- path is user's own CLI argument
	if err != nil {
		return "", "", fmt.Errorf("failed to read file: %w", err)
	}

	return string(data), absPath, nil
}

// updateStatusBar refreshes the status bar with current viewer state.
func updateStatusBar(statusBar *tview.TextView, v *tviewAdapter.TextViewViewer) {
	core := v.Core()
	srcPath := core.SourceFilePath()
	fileName := filepath.Base(srcPath)
	if fileName == "" || fileName == "." {
		fileName = "navidown"
	}

	// history indicators
	canBack := core.CanGoBack()
	canForward := core.CanGoForward()

	keyColor := "gray"
	activeColor := "white"
	status := fmt.Sprintf(" [yellow]%s[-] | Link:[%s]Tab/Shift-Tab[-] | Back:", fileName, keyColor)
	if canBack {
		status += fmt.Sprintf("[%s]◀[-]", activeColor)
	} else {
		status += "[gray]◀[-]"
	}
	status += " Fwd:"
	if canForward {
		status += fmt.Sprintf("[%s]▶[-]", activeColor)
	} else {
		status += "[gray]▶[-]"
	}
	status += fmt.Sprintf(" | Scroll:[%s]j/k[-] Top/End:[%s]g/G[-] Quit:[%s]q[-]", keyColor, keyColor, keyColor)

	statusBar.SetText(status)
}

// splitFragment separates a URL into path and fragment components.
func splitFragment(url string) (path, fragment string) {
	path, fragment, _ = strings.Cut(url, "#")
	return path, fragment
}
