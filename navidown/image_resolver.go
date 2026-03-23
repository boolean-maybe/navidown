package navidown

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"  // Register standard image decoders
	_ "image/jpeg" // Register standard image decoders
	"image/png"
	"io"
	"math"
	"net/http"
	"os"
	"sync"

	_ "golang.org/x/image/bmp"  // Register BMP decoder
	_ "golang.org/x/image/tiff" // Register TIFF decoder
	_ "golang.org/x/image/webp" // Register WebP decoder
)

// ImageInfo holds decoded image metadata and raw bytes.
type ImageInfo struct {
	Width  int    // Pixel width
	Height int    // Pixel height
	Format string // "png", "jpeg", "gif", etc.
	Data   []byte // Raw image bytes (for transmission)
}

const defaultSVGRasterWidth = 2048

const defaultSVGScaleFactor = 1

// ImageResolver fetches images and decodes their dimensions.
type ImageResolver struct {
	searchRoots    []string
	cache          sync.Map // url -> *ImageInfo
	svgRasterizer  SVGRasterizer
	svgRasterWidth int
	svgScaleFactor int
}

// NewImageResolver creates a new resolver.
func NewImageResolver(searchRoots []string) *ImageResolver {
	return &ImageResolver{
		searchRoots: searchRoots,
	}
}

// SetSVGRasterizer overrides the default ResvgRasterizer (useful for testing).
func (r *ImageResolver) SetSVGRasterizer(rast SVGRasterizer) {
	r.svgRasterizer = rast
}

// SetSVGRasterWidth sets the fallback width in pixels used when rasterizing
// SVGs that have no intrinsic dimensions. Zero means use the default (2048).
func (r *ImageResolver) SetSVGRasterWidth(px int) {
	r.svgRasterWidth = px
}

// SetSVGScaleFactor sets the multiplier applied to SVG intrinsic dimensions
// when rasterizing. For example, 2 means a 90px-wide badge is rasterized at
// 180px for HiDPI clarity. Zero means use the default (2).
func (r *ImageResolver) SetSVGScaleFactor(f int) {
	r.svgScaleFactor = f
}

// Resolve fetches an image and returns its info. Results are cached.
func (r *ImageResolver) Resolve(url string, sourceFilePath string) (*ImageInfo, error) {
	// Check cache first
	if cached, ok := r.cache.Load(url); ok {
		info, _ := cached.(*ImageInfo)
		return info, nil
	}

	data, err := r.fetchImageBytes(url, sourceFilePath)
	if err != nil {
		return nil, err
	}

	if isSVGData(data) {
		data, err = r.rasterizeSVG(data)
		if err != nil {
			return nil, fmt.Errorf("rasterize SVG %q: %w", url, err)
		}
	}

	info, err := decodeImageInfo(data)
	if err != nil {
		return nil, fmt.Errorf("decode image %q: %w", url, err)
	}

	r.cache.Store(url, info)
	return info, nil
}

// PreResolve resolves multiple image URLs in parallel, populating the cache.
// Errors are silently ignored — individual Resolve calls will re-attempt.
func (r *ImageResolver) PreResolve(urls []string, sourceFilePath string) {
	if len(urls) == 0 {
		return
	}

	var wg sync.WaitGroup
	for _, u := range urls {
		// skip already-cached URLs
		if _, ok := r.cache.Load(u); ok {
			continue
		}
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			_, _ = r.Resolve(url, sourceFilePath)
		}(u)
	}
	wg.Wait()
}

// Close releases resources held by the resolver (e.g. caching rasterizer temp dirs).
func (r *ImageResolver) Close() {
	if c, ok := r.svgRasterizer.(*CachingSVGRasterizer); ok {
		c.Close()
	}
}

func (r *ImageResolver) rasterizeSVG(data []byte) ([]byte, error) {
	rast := r.svgRasterizer
	if rast == nil {
		rast = r.defaultRasterizer()
	}

	width := r.svgRasterizeWidth(data)
	return rast.Rasterize(data, width)
}

// svgRasterizeWidth determines the pixel width to rasterize an SVG at.
// If the SVG has intrinsic dimensions, uses intrinsicWidth * scaleFactor.
// Otherwise falls back to svgRasterWidth (default 2048).
func (r *ImageResolver) svgRasterizeWidth(data []byte) int {
	if w, _, ok := parseSVGDimensions(data); ok && w > 0 {
		scale := r.svgScaleFactor
		if scale <= 0 {
			scale = defaultSVGScaleFactor
		}
		return int(math.Ceil(w)) * scale
	}

	fallback := r.svgRasterWidth
	if fallback <= 0 {
		fallback = defaultSVGRasterWidth
	}
	return fallback
}

// defaultRasterizer creates and caches a CachingSVGRasterizer (or falls back to bare ResvgRasterizer).
func (r *ImageResolver) defaultRasterizer() SVGRasterizer {
	rast := NewCachingSVGRasterizer(&ResvgRasterizer{}, "")
	if rast != nil {
		r.svgRasterizer = rast
		return rast
	}
	bare := &ResvgRasterizer{}
	r.svgRasterizer = bare
	return bare
}

func (r *ImageResolver) fetchImageBytes(url string, sourceFilePath string) ([]byte, error) {
	if isHTTPURL(url) {
		return r.fetchHTTP(url)
	}

	resolved, err := ResolveMarkdownPath(url, sourceFilePath, r.searchRoots)
	if err != nil {
		return nil, fmt.Errorf("resolve image path %q: %w", url, err)
	}

	if isHTTPURL(resolved) {
		return r.fetchHTTP(resolved)
	}

	return os.ReadFile(resolved)
}

func (r *ImageResolver) fetchHTTP(url string) ([]byte, error) {
	resp, err := http.Get(url) //nolint:gosec // URL is user-provided, already validated by path resolver
	if err != nil {
		return nil, fmt.Errorf("fetch image: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch image: HTTP %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func decodeImageInfo(data []byte) (*ImageInfo, error) {
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	return &ImageInfo{
		Width:  config.Width,
		Height: config.Height,
		Format: format,
		Data:   data,
	}, nil
}

// ResizedPNG returns the image data resized to the given pixel dimensions,
// encoded as PNG. Uses nearest-neighbor scaling (fast, no external deps).
// Always resizes when dimensions don't exactly match the target to ensure
// pixel-perfect alignment with the terminal's cell grid and avoid seam artifacts.
func (info *ImageInfo) ResizedPNG(targetW, targetH int) ([]byte, error) {
	if targetW <= 0 || targetH <= 0 {
		return info.Data, nil
	}
	if info.Width == targetW && info.Height == targetH {
		return info.Data, nil
	}

	src, _, err := image.Decode(bytes.NewReader(info.Data))
	if err != nil {
		return nil, fmt.Errorf("decode for resize: %w", err)
	}

	dst := image.NewRGBA(image.Rect(0, 0, targetW, targetH))
	srcBounds := src.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()

	// Nearest-neighbor resize
	for y := 0; y < targetH; y++ {
		srcY := srcBounds.Min.Y + y*srcH/targetH
		for x := 0; x < targetW; x++ {
			srcX := srcBounds.Min.X + x*srcW/targetW
			dst.Set(x, y, src.At(srcX, srcY))
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, dst); err != nil {
		return nil, fmt.Errorf("encode resized PNG: %w", err)
	}
	return buf.Bytes(), nil
}

// CellDimensions calculates how many terminal columns and rows an image
// should occupy given the available width and an assumed cell aspect ratio.
// cellWidth and cellHeight are in pixels (typical: 8x16 or 10x20).
// maxCols is the maximum number of columns available.
// maxRows caps the vertical size; if the image would exceed maxRows,
// columns are scaled down proportionally. Use 0 for no row limit.
func CellDimensions(imgWidth, imgHeight, maxCols, cellWidth, cellHeight, maxRows int) (cols, rows int) {
	if imgWidth <= 0 || imgHeight <= 0 || maxCols <= 0 || cellWidth <= 0 || cellHeight <= 0 {
		return 0, 0
	}

	// Calculate the number of columns needed at native resolution
	nativeCols := (imgWidth + cellWidth - 1) / cellWidth

	// Cap to available width
	cols = min(nativeCols, maxCols)

	// Calculate rows maintaining aspect ratio.
	// Floor instead of rounding so the placeholder box does not overshoot the
	// image aspect ratio and trigger extra padding in unicode placements.
	// pixels per column = imgWidth / cols
	// total height in pixels at this column count = imgHeight * (cols * cellWidth) / imgWidth
	scaledHeight := float64(imgHeight) * float64(cols) * float64(cellWidth) / float64(imgWidth)
	rows = max(int(scaledHeight/float64(cellHeight)), 1)

	// If rows exceed the cap, scale columns down to fit maxRows
	if maxRows > 0 && rows > maxRows {
		rows = maxRows
		// Back-calculate columns from maxRows, preserving aspect ratio
		// maxRows * cellHeight pixels of height → how many columns?
		scaledWidth := float64(imgWidth) * float64(rows) * float64(cellHeight) / float64(imgHeight)
		cols = max(int(scaledWidth/float64(cellWidth)), 1)
		cols = min(cols, maxCols)
	}

	return cols, rows
}
