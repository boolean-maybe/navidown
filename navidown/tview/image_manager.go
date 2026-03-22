package tview

import (
	"encoding/base64"
	"fmt"
	"io"
	"strings"
	"sync"

	nav "github.com/boolean-maybe/navidown/navidown"
	"github.com/gdamore/tcell/v2"
)

// imageDisplay holds the computed cell grid for a resolved image.
type imageDisplay struct {
	cols, rows int // cell grid dimensions
	maxCols    int // viewport width used for layout (needed for recomputation)
}

// ImageManager handles Kitty image protocol transmission and lifecycle.
// It tracks which images have been transmitted to the terminal and manages
// ID allocation for the Unicode placeholder protocol.
type ImageManager struct {
	mu          sync.Mutex
	nextID      uint32
	transmitted map[uint32]bool // image IDs that have been sent to the terminal
	urlToID     map[string]uint32
	idToInfo    map[uint32]*nav.ImageInfo
	idToDisplay map[uint32]imageDisplay
	supported   *bool // nil = unknown, pointer to bool = detected
	purgedStale bool  // true after initial purge of stale Kitty images
	resolver    *nav.ImageResolver
	cellWidth   int
	cellHeight  int
	maxRows     int // maximum rows for an image (0 = no limit)
}

// NewImageManager creates a new image manager.
// cellWidth and cellHeight are the assumed pixel dimensions of a terminal cell.
// Common defaults: 8x16 for standard terminals, 10x20 for HiDPI.
func NewImageManager(resolver *nav.ImageResolver, cellWidth, cellHeight int) *ImageManager {
	if cellWidth <= 0 {
		cellWidth = 8
	}
	if cellHeight <= 0 {
		cellHeight = 16
	}
	return &ImageManager{
		// Start IDs at 0x01010101 so all 4 bytes are non-zero.
		// The upper byte is encoded as the 3rd diacritic in Unicode placeholders,
		// and the lower 3 bytes are encoded in the RGB foreground color.
		// Non-zero bytes in all positions ensure Kitty correctly associates
		// placeholder cells with their image.
		nextID:      0x01010101,
		transmitted: make(map[uint32]bool),
		urlToID:     make(map[string]uint32),
		idToInfo:    make(map[uint32]*nav.ImageInfo),
		idToDisplay: make(map[uint32]imageDisplay),
		resolver:    resolver,
		cellWidth:   cellWidth,
		cellHeight:  cellHeight,
		maxRows:     0,
	}
}

// SetMaxRows sets the maximum number of terminal rows an image can occupy.
// Use 0 for no limit. Default is 0 (no limit).
func (m *ImageManager) SetMaxRows(maxRows int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.maxRows = maxRows
}

// UpdateCellSize queries the terminal for actual cell pixel dimensions and
// updates the manager if they differ from the current values. When cell size
// changes, cached display dimensions and transmission state are invalidated
// so images are re-resolved and re-transmitted at the correct size.
// Returns true if cell size changed, false otherwise.
func (m *ImageManager) UpdateCellSize(screen tcell.Screen) bool {
	tty, ok := screen.Tty()
	if !ok {
		return false
	}
	ws, err := tty.WindowSize()
	if err != nil {
		return false
	}
	cellW, cellH := ws.CellDimensions()
	if cellW <= 0 || cellH <= 0 {
		return false
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if cellW == m.cellWidth && cellH == m.cellHeight {
		return false
	}

	m.cellWidth = cellW
	m.cellHeight = cellH

	// Recompute display dimensions for all known images and clear
	// transmission state so images are re-transmitted at the correct size.
	for id, disp := range m.idToDisplay {
		info := m.idToInfo[id]
		if info == nil {
			continue
		}
		cols, rows := nav.CellDimensions(info.Width, info.Height, disp.maxCols, cellW, cellH, m.maxRows)
		m.idToDisplay[id] = imageDisplay{
			cols:    cols,
			rows:    rows,
			maxCols: disp.maxCols,
		}
	}
	m.transmitted = make(map[uint32]bool)
	return true
}

// Supported returns whether the terminal supports Kitty graphics protocol.
// Returns false if detection hasn't run or the terminal doesn't support it.
func (m *ImageManager) Supported() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.supported == nil {
		return false
	}
	return *m.supported
}

// SetSupported explicitly sets whether image support is available.
// Call this after terminal capability detection.
func (m *ImageManager) SetSupported(supported bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.supported = &supported
}

// DetectSupport queries the terminal for Kitty graphics protocol support.
// This sends a test graphics command and checks for a response.
// Must be called with access to a Tty that supports reading responses.
func (m *ImageManager) DetectSupport(screen tcell.Screen) {
	tty := extractTty(screen)
	if tty == nil {
		m.SetSupported(false)
		return
	}

	// Send a query: transmit a 1x1 test image with query action
	// a=q means query (don't display), t=d means direct data, f=24 means RGB
	query := "\x1b_Ga=q,i=31,f=24,s=1,v=1,t=d;AAAA\x1b\\"
	if _, err := io.WriteString(tty, query); err != nil {
		m.SetSupported(false)
		return
	}

	// For now, assume support if we can write to the tty.
	// Full detection would require reading the response, which is complex
	// with tcell's event loop. We'll rely on explicit opt-in or env vars.
	m.SetSupported(true)
}

// PreResolveImages resolves multiple image URLs in parallel, warming the cache.
// Subsequent ResolveAndAllocate calls for these URLs will be fast cache hits.
func (m *ImageManager) PreResolveImages(urls []string, sourceFilePath string) {
	m.resolver.PreResolve(urls, sourceFilePath)
}

// ResolveAndAllocate resolves an image URL and assigns it a Kitty image ID.
// Returns the ImagePlaceholder with dimensions calculated for the given viewport width.
func (m *ImageManager) ResolveAndAllocate(url, sourceFilePath string, maxCols int) (*nav.ImagePlaceholder, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if we already have an ID for this URL
	if id, ok := m.urlToID[url]; ok {
		info := m.idToInfo[id]
		cols, rows := nav.CellDimensions(info.Width, info.Height, maxCols, m.cellWidth, m.cellHeight, m.maxRows)
		m.idToDisplay[id] = imageDisplay{
			cols:    cols,
			rows:    rows,
			maxCols: maxCols,
		}
		return &nav.ImagePlaceholder{
			ImageID: id,
			Cols:    cols,
			Rows:    rows,
			URL:     url,
		}, nil
	}

	// Resolve the image
	info, err := m.resolver.Resolve(url, sourceFilePath)
	if err != nil {
		return nil, err
	}

	// Allocate an ID
	id := m.nextID
	m.nextID++
	m.urlToID[url] = id
	m.idToInfo[id] = info

	cols, rows := nav.CellDimensions(info.Width, info.Height, maxCols, m.cellWidth, m.cellHeight, m.maxRows)

	// Store display dimensions for the virtual placement.
	m.idToDisplay[id] = imageDisplay{
		cols:    cols,
		rows:    rows,
		maxCols: maxCols,
	}

	return &nav.ImagePlaceholder{
		ImageID: id,
		Cols:    cols,
		Rows:    rows,
		URL:     url,
	}, nil
}

// EnsureTransmitted sends image data to the terminal if not already sent.
// Must be called during Draw() when we have access to the screen.
func (m *ImageManager) EnsureTransmitted(screen tcell.Screen, id uint32) error {
	m.mu.Lock()
	// On first transmission, purge any stale images left in Kitty's cache
	// from previous runs. Without this, Kitty may serve a cached image under
	// the same ID instead of accepting the newly transmitted data.
	if !m.purgedStale {
		m.purgedStale = true
		m.mu.Unlock()
		m.DeleteAll(screen)
		m.mu.Lock()
	}

	if m.transmitted[id] {
		m.mu.Unlock()
		return nil
	}

	info, ok := m.idToInfo[id]
	display := m.idToDisplay[id]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("unknown image ID %d", id)
	}
	m.mu.Unlock()

	cols, rows := display.cols, display.rows

	tty := extractTty(screen)
	if tty == nil {
		return fmt.Errorf("no tty available for image transmission")
	}

	// Transmit with a=T,U=1 (combined transmit+display with unicode placement).
	// Kitty handles scaling via c=/r= — no pre-resize needed.
	if err := transmitImage(tty, id, info.Data, cols, rows); err != nil {
		return err
	}

	m.mu.Lock()
	m.transmitted[id] = true
	m.mu.Unlock()
	return nil
}

// DeleteImage removes an image from the terminal and internal tracking.
func (m *ImageManager) DeleteImage(screen tcell.Screen, id uint32) {
	m.mu.Lock()
	delete(m.transmitted, id)
	m.mu.Unlock()

	tty := extractTty(screen)
	if tty == nil {
		return
	}

	// Delete by image ID
	cmd := fmt.Sprintf("\x1b_Ga=d,d=I,i=%d\x1b\\", id)
	_, _ = io.WriteString(tty, cmd)
}

// DeleteAll removes all images from the terminal.
func (m *ImageManager) DeleteAll(screen tcell.Screen) {
	m.mu.Lock()
	m.transmitted = make(map[uint32]bool)
	m.mu.Unlock()

	tty := extractTty(screen)
	if tty == nil {
		return
	}

	// Delete all placements
	cmd := "\x1b_Ga=d,d=A\x1b\\"
	_, _ = io.WriteString(tty, cmd)
}

// BuildPlaceholderLines generates the Unicode placeholder lines for a resolved image.
// Each line is a string of tview-tagged text containing U+10EEEE characters
// with combining diacritics encoding row/column/id and foreground color encoding image ID.
//
// Each placeholder cell has 3 combining marks matching icat's protocol:
//  1. Row diacritic (encodes the row index)
//  2. Column diacritic (encodes the column index)
//  3. ID diacritic (encodes the upper byte of the image ID)
//
// The lower 3 bytes of the image ID are encoded in the foreground color (RGB).
func BuildPlaceholderLines(placeholder *nav.ImagePlaceholder) []string {
	if placeholder.Cols <= 0 || placeholder.Rows <= 0 {
		return nil
	}

	colorTag := "[" + nav.ImageIDToColor(placeholder.ImageID) + "]"
	resetTag := "[-]"

	// 3rd diacritic encodes the upper byte (byte 3) of the image ID.
	// Only emitted when the upper byte is non-zero (ID > 0xFFFFFF).
	upperByte := (placeholder.ImageID >> 24) & 0xFF
	var idDiacritic rune
	if upperByte > 0 && int(upperByte) <= maxDiacriticIndex {
		idDiacritic = rowColumnDiacritics[upperByte]
	}

	lines := make([]string, placeholder.Rows)
	for row := 0; row < placeholder.Rows; row++ {
		var sb strings.Builder
		sb.WriteString(colorTag)
		for col := 0; col < placeholder.Cols; col++ {
			sb.WriteRune(placeholderRune)
			if row <= maxDiacriticIndex {
				sb.WriteRune(rowColumnDiacritics[row])
			}
			if col <= maxDiacriticIndex {
				sb.WriteRune(rowColumnDiacritics[col])
			}
			if idDiacritic != 0 {
				sb.WriteRune(idDiacritic)
			}
		}
		sb.WriteString(resetTag)
		lines[row] = sb.String()
	}

	return lines
}

// transmitImage sends image data to the terminal using Kitty graphics protocol.
// Uses a=T,U=1 to transmit and create a virtual/unicode placement in one command,
// with c=/r= so Kitty handles scaling on the GPU. This matches how icat works
// with --unicode-placeholder.
func transmitImage(w io.Writer, id uint32, data []byte, cols, rows int) error {
	encoded := base64.StdEncoding.EncodeToString(data)

	// Kitty protocol uses chunked transmission for data > 4096 bytes
	const chunkSize = 4096

	for i := 0; i < len(encoded); i += chunkSize {
		end := min(i+chunkSize, len(encoded))
		chunk := encoded[i:end]

		isFirst := i == 0
		isLast := end >= len(encoded)

		var cmd string
		if isFirst && isLast {
			// a=T: transmit and display; U=1: unicode/virtual placement;
			// c=/r=: cell grid dimensions for scaling; f=100: auto-detect format
			cmd = fmt.Sprintf("\x1b_Ga=T,q=2,U=1,i=%d,f=100,t=d,c=%d,r=%d;%s\x1b\\", id, cols, rows, chunk)
		} else if isFirst {
			cmd = fmt.Sprintf("\x1b_Ga=T,q=2,U=1,i=%d,f=100,t=d,c=%d,r=%d,m=1;%s\x1b\\", id, cols, rows, chunk)
		} else if isLast {
			cmd = fmt.Sprintf("\x1b_Gm=0;%s\x1b\\", chunk)
		} else {
			cmd = fmt.Sprintf("\x1b_Gm=1;%s\x1b\\", chunk)
		}

		if _, err := io.WriteString(w, cmd); err != nil {
			return fmt.Errorf("transmit image chunk: %w", err)
		}
	}

	return nil
}

// extractTty gets the underlying tty writer from a tcell.Screen.
func extractTty(screen tcell.Screen) io.Writer {
	tty, ok := screen.Tty()
	if !ok {
		return nil
	}
	return tty
}
