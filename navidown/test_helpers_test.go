package navidown

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
