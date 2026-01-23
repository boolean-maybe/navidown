package ansi

import (
	"regexp"
	"strings"
	"text/template"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// TemplateFuncMap contains a few useful template helpers.
var (
	TemplateFuncMap = template.FuncMap{
		"Left": func(values ...interface{}) string {
			if len(values) < 2 { //nolint:mnd
				return ""
			}
			s, ok := values[0].(string)
			if !ok {
				return ""
			}
			n, ok := values[1].(int)
			if !ok {
				return ""
			}
			if n > len(s) {
				n = len(s)
			}

			return s[:n]
		},
		"Matches": func(values ...interface{}) bool {
			if len(values) < 2 { //nolint:mnd
				return false
			}
			s, ok := values[0].(string)
			if !ok {
				return false
			}
			pattern, ok := values[1].(string)
			if !ok {
				return false
			}
			matched, err := regexp.MatchString(pattern, s)
			return err == nil && matched
		},
		"Mid": func(values ...interface{}) string {
			if len(values) < 2 { //nolint:mnd
				return ""
			}
			s, ok := values[0].(string)
			if !ok {
				return ""
			}
			l, ok := values[1].(int)
			if !ok {
				return ""
			}
			if l > len(s) {
				l = len(s)
			}

			if len(values) > 2 { //nolint:mnd
				r, ok := values[2].(int)
				if !ok {
					return ""
				}
				if r > len(s) {
					r = len(s)
				}
				return s[l:r]
			}
			return s[l:]
		},
		"Right": func(values ...interface{}) string {
			if len(values) < 2 { //nolint:mnd
				return ""
			}
			s, ok := values[0].(string)
			if !ok {
				return ""
			}
			n, ok := values[1].(int)
			if !ok {
				return ""
			}
			n = len(s) - n
			if n < 0 {
				n = 0
			}

			return s[n:]
		},
		"Last": func(values ...interface{}) string {
			if len(values) < 1 {
				return ""
			}
			parts, ok := values[0].([]string)
			if !ok || len(parts) == 0 {
				return ""
			}
			return parts[len(parts)-1]
		},
		// strings functions
		"Compare":      strings.Compare, // 1.5+ only
		"Contains":     strings.Contains,
		"ContainsAny":  strings.ContainsAny,
		"Count":        strings.Count,
		"EqualFold":    strings.EqualFold,
		"HasPrefix":    strings.HasPrefix,
		"HasSuffix":    strings.HasSuffix,
		"Index":        strings.Index,
		"IndexAny":     strings.IndexAny,
		"Join":         strings.Join,
		"LastIndex":    strings.LastIndex,
		"LastIndexAny": strings.LastIndexAny,
		"Repeat":       strings.Repeat,
		"Replace":      strings.Replace,
		"Split":        strings.Split,
		"SplitAfter":   strings.SplitAfter,
		"SplitAfterN":  strings.SplitAfterN,
		"SplitN":       strings.SplitN,
		"Title":        cases.Title(language.English).String,
		"ToLower":      cases.Lower(language.English).String,
		"ToTitle":      cases.Upper(language.English).String,
		"ToUpper":      strings.ToUpper,
		"Trim":         strings.Trim,
		"TrimLeft":     strings.TrimLeft,
		"TrimPrefix":   strings.TrimPrefix,
		"TrimRight":    strings.TrimRight,
		"TrimSpace":    strings.TrimSpace,
		"TrimSuffix":   strings.TrimSuffix,
	}
)
