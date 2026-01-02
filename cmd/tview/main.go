package main

import (
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
	// Parse arguments
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <file-path-or-url>\n", os.Args[0])
		os.Exit(1)
	}

	arg := os.Args[1]

	// Load initial content
	content, sourcePath, err := loadContent(arg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading content: %v\n", err)
		os.Exit(1)
	}

	// Create TView application
	app := tview.NewApplication()

	// Create markdown viewer
	mdViewer := tviewAdapter.New()

	// Set custom ANSI converter for proper background color support
	mdViewer.SetAnsiConverter(util.NewAnsiConverter(true))

	// Set up content fetcher for link navigation
	provider := &loaders.FileHTTP{SearchRoots: []string{"."}}

	// Wire up link activation handler - manually fetch and update through adapter
	mdViewer.SetSelectHandler(func(v *tviewAdapter.Viewer, elem navidown.NavElement) {
		if elem.Type != navidown.NavElementURL {
			return
		}

		// Fetch content
		content, err := provider.FetchContent(elem)
		if err != nil {
			errorContent := "# Error\n\nFailed to load `" + elem.URL + "`:\n\n```\n" + err.Error() + "\n```"
			v.SetMarkdownWithSource(errorContent, elem.SourceFilePath, true)
			return
		}

		if content == "" {
			return
		}

		// Resolve path
		newSourcePath := elem.URL
		if !strings.HasPrefix(elem.URL, "http://") && !strings.HasPrefix(elem.URL, "https://") && elem.SourceFilePath != "" {
			resolved, rerr := navidown.ResolveMarkdownPath(elem.URL, elem.SourceFilePath, []string{"."})
			if rerr == nil && resolved != "" {
				newSourcePath = resolved
			}
		}

		// Update through adapter (this will refresh display)
		v.SetMarkdownWithSource(content, newSourcePath, true)
	})

	// Create status bar
	statusBar := tview.NewTextView()
	statusBar.SetDynamicColors(true)
	statusBar.SetTextAlign(tview.AlignLeft)

	// Update status bar on state changes
	updateStatusBar := func(v *tviewAdapter.Viewer) {
		core := v.Core()
		srcPath := core.SourceFilePath()
		fileName := filepath.Base(srcPath)
		if fileName == "" {
			fileName = "navidown"
		}

		// Count links
		linkCount := 0
		for _, elem := range core.Elements() {
			if elem.Type == navidown.NavElementURL {
				linkCount++
			}
		}

		// History indicators
		canBack := core.CanGoBack()
		canForward := core.CanGoForward()

		status := fmt.Sprintf(" [yellow]%s[-] | Links: %d | ", fileName, linkCount)
		if canBack {
			status += "[green]◀[-] "
		} else {
			status += "[gray]◀[-] "
		}
		if canForward {
			status += "[green]▶[-] "
		} else {
			status += "[gray]▶[-] "
		}
		status += "| q:quit"

		statusBar.SetText(status)
	}

	mdViewer.SetStateChangedHandler(updateStatusBar)

	// Load initial content
	mdViewer.SetMarkdownWithSource(content, sourcePath, false)

	// Initial status bar update
	updateStatusBar(mdViewer)

	// Create flex layout with status bar
	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(mdViewer, 0, 1, true).
		AddItem(statusBar, 1, 0, false)

	// Set up quit handler
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 'q' {
			app.Stop()
			return nil
		}
		return event
	})

	// Run application
	if err := app.SetRoot(flex, true).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running application: %v\n", err)
		os.Exit(1)
	}
}

// loadContent loads content from a file path or URL
func loadContent(arg string) (content string, sourcePath string, err error) {
	// Check if it's an HTTP(S) URL
	if strings.HasPrefix(arg, "http://") || strings.HasPrefix(arg, "https://") {
		provider := &loaders.FileHTTP{}
		content, err := provider.FetchContent(navidown.NavElement{URL: arg})
		return content, arg, err
	}

	// Local file
	absPath, err := filepath.Abs(arg)
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve path: %w", err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to read file: %w", err)
	}

	return string(data), absPath, nil
}
