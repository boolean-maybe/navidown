package navidown

import "errors"

type staticRenderer struct {
	lines   []string
	cleaner LineCleaner
}

func (r staticRenderer) Render(_ string) (RenderResult, error) {
	cleaner := r.cleaner
	if cleaner == nil {
		cleaner = LineCleanerFunc(func(s string) string { return s })
	}
	return RenderResult{Lines: r.lines, Cleaner: cleaner}, nil
}

// errorRenderer always returns an error, used for testing error handling.
type errorRenderer struct {
	err error
}

func (r errorRenderer) Render(_ string) (RenderResult, error) {
	return RenderResult{}, r.err
}

var errRenderFailed = errors.New("render failed")
