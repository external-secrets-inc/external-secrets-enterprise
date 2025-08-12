package github

import "strings"

type pathFilter struct {
	exact    map[string]struct{} // exact file matches: "a/b/c.txt"
	prefixes []string            // directory prefixes: "a/b/" (must end with '/')
}

func newPathFilter(paths []string) *pathFilter {
	f := &pathFilter{
		exact:    make(map[string]struct{}),
		prefixes: make([]string, 0, len(paths)),
	}

	seenPrefix := make(map[string]struct{})

	for _, p := range paths {
		p = strings.TrimSpace(p)
		p = strings.TrimPrefix(p, "/")
		if p == "" {
			continue
		}

		f.exact[p] = struct{}{}

		pref := p
		if !strings.HasSuffix(pref, "/") {
			pref += "/"
		}
		if _, dup := seenPrefix[pref]; !dup {
			f.prefixes = append(f.prefixes, pref)
			seenPrefix[pref] = struct{}{}
		}
	}

	return f
}

func (f *pathFilter) allow(path string) bool {
	// No filters -> allow all
	if len(f.exact) == 0 && len(f.prefixes) == 0 {
		return true
	}
	if _, ok := f.exact[path]; ok {
		return true
	}
	for _, pre := range f.prefixes {
		if strings.HasPrefix(path, pre) {
			return true
		}
	}
	return false
}
