package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/boolean-maybe/navidown/loaders"
	"github.com/boolean-maybe/navidown/navidown"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	viewport viewport.Model
	viewer   *navidown.Viewer
	fetcher  *navidown.ContentFetcher
	ready    bool
	width    int
	height   int
}

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

	// Create core viewer
	viewer := navidown.New(navidown.Options{
		Renderer:   navidown.NewANSIRenderer(),
		Correlator: navidown.NewScoringCorrelator(),
		HistoryMax: 50,
	})

	// Load initial content
	viewer.SetMarkdownWithSource(content, sourcePath, false)

	// Create content fetcher
	fetcher := navidown.NewContentFetcher(
		&loaders.FileHTTP{SearchRoots: []string{"."}},
		[]string{"."},
	)

	// Create initial model
	m := model{
		viewer:  viewer,
		fetcher: fetcher,
		ready:   false,
	}

	// Run Bubble Tea program
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running application: %v\n", err)
		os.Exit(1)
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		if !m.ready {
			// Initialize viewport on first resize
			m.viewport = viewport.New(msg.Width, msg.Height-1) // -1 for status bar
			m.viewport.YPosition = 0
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 1
		}

		// Update content
		m.syncViewportContent()

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "up", "k":
			if m.viewer.ScrollUp(m.viewport.Height) {
				m.syncViewportContent()
			}

		case "down", "j":
			if m.viewer.ScrollDown(m.viewport.Height) {
				m.syncViewportContent()
			}

		case "pgup":
			if m.viewer.PageUp(m.viewport.Height) {
				m.syncViewportContent()
			}

		case "pgdown":
			if m.viewer.PageDown(m.viewport.Height) {
				m.syncViewportContent()
			}

		case "home", "g":
			m.viewer.Home(m.viewport.Height)
			m.syncViewportContent()

		case "end", "G":
			m.viewer.End(m.viewport.Height)
			m.syncViewportContent()

		case "tab":
			if m.viewer.MoveToNextLink(m.viewport.Height) {
				m.syncViewportContent()
			}

		case "shift+tab":
			if m.viewer.MoveToPreviousLink(m.viewport.Height) {
				m.syncViewportContent()
			}

		case "enter":
			if sel := m.viewer.Selected(); sel != nil && sel.Type == navidown.NavElementURL {
				m.fetcher.OnSelect(m.viewer, *sel)
				m.syncViewportContent()
			}

		case "alt+left":
			if m.viewer.GoBack() {
				m.syncViewportContent()
			}

		case "alt+right":
			if m.viewer.GoForward() {
				m.syncViewportContent()
			}
		}
	}

	// Update viewport
	m.viewport, cmd = m.viewport.Update(msg)

	return m, cmd
}

func (m model) View() string {
	if !m.ready {
		return "Loading..."
	}

	// Render viewport content
	content := m.viewport.View()

	// Render status bar
	statusBar := m.renderStatusBar()

	return content + "\n" + statusBar
}

// syncViewportContent updates the viewport with current viewer state
func (m *model) syncViewportContent() {
	lines := m.viewer.RenderedLines()

	// Add selection highlighting
	if sel := m.viewer.Selected(); sel != nil && sel.Type == navidown.NavElementURL {
		lines = m.addSelectionHighlight(lines, sel)
	}

	// Join lines and set viewport content
	content := strings.Join(lines, "\n")
	m.viewport.SetContent(content)

	// Sync scroll position
	m.viewport.YOffset = m.viewer.ScrollOffset()
}

// addSelectionHighlight adds ANSI reverse video codes to highlight selected link
func (m *model) addSelectionHighlight(lines []string, sel *navidown.NavElement) []string {
	if sel.StartLine < 0 || sel.StartLine >= len(lines) {
		return lines
	}

	line := lines[sel.StartLine]
	runes := []rune(line)

	// Strip ANSI codes to find actual character positions
	cleanLine := stripANSI(line)
	cleanRunes := []rune(cleanLine)

	if sel.StartCol < 0 || sel.StartCol >= len(cleanRunes) || sel.EndCol <= sel.StartCol {
		return lines
	}

	// Build new line with highlighting
	// This is complex because we need to insert ANSI codes while preserving existing ones
	// For now, use a simpler approach: rebuild line with reverse video
	var result strings.Builder
	col := 0
	inEscape := false

	for i := 0; i < len(runes); i++ {
		r := runes[i]

		// Track ANSI escape sequences
		if r == '\x1b' && i+1 < len(runes) && runes[i+1] == '[' {
			inEscape = true
		}

		if inEscape {
			result.WriteRune(r)
			if r == 'm' {
				inEscape = false
			}
			continue
		}

		// Add reverse video at start of selection
		if col == sel.StartCol {
			result.WriteString("\x1b[7m") // Reverse video on
		}

		result.WriteRune(r)
		col++

		// Remove reverse video at end of selection
		if col == sel.EndCol {
			result.WriteString("\x1b[27m") // Reverse video off
		}
	}

	// Create new lines slice with modified line
	newLines := make([]string, len(lines))
	copy(newLines, lines)
	newLines[sel.StartLine] = result.String()

	return newLines
}

// stripANSI removes ANSI escape sequences from a string
func stripANSI(s string) string {
	var result strings.Builder
	runes := []rune(s)
	i := 0

	for i < len(runes) {
		if runes[i] == '\x1b' && i+1 < len(runes) && runes[i+1] == '[' {
			// Skip until 'm'
			i += 2
			for i < len(runes) && runes[i] != 'm' {
				i++
			}
			i++ // Skip 'm'
			continue
		}
		result.WriteRune(runes[i])
		i++
	}

	return result.String()
}

// renderStatusBar renders the status bar at the bottom
func (m model) renderStatusBar() string {
	srcPath := m.viewer.SourceFilePath()
	fileName := filepath.Base(srcPath)
	if fileName == "" {
		fileName = "navidown"
	}

	// Count links
	linkCount := 0
	for _, elem := range m.viewer.Elements() {
		if elem.Type == navidown.NavElementURL {
			linkCount++
		}
	}

	// History indicators
	backSymbol := "◀"
	forwardSymbol := "▶"
	backColor := lipgloss.Color("240")
	forwardColor := lipgloss.Color("240")

	if m.viewer.CanGoBack() {
		backColor = lipgloss.Color("green")
	}
	if m.viewer.CanGoForward() {
		forwardColor = lipgloss.Color("green")
	}

	fileStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("yellow"))
	linkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("cyan"))
	backStyle := lipgloss.NewStyle().Foreground(backColor)
	forwardStyle := lipgloss.NewStyle().Foreground(forwardColor)
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	statusBar := lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("255")).
		Width(m.width).
		Render(
			fmt.Sprintf(" %s | Links: %s | %s %s | %s",
				fileStyle.Render(fileName),
				linkStyle.Render(fmt.Sprint(linkCount)),
				backStyle.Render(backSymbol),
				forwardStyle.Render(forwardSymbol),
				helpStyle.Render("q:quit"),
			),
		)

	return statusBar
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
