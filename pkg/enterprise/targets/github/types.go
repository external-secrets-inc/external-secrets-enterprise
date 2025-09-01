// Copyright External Secrets Inc. 2025
// All Rights Reserved

package github

import "strings"

type pathFilter struct {
	exact    map[string]struct{} // exact file matches: "a/b/c.txt"
	prefixes []string            // directory prefixes: "a/b/" (must end with '/')
}

func newPathFilter(paths []string) *pathFilter {
	filter := &pathFilter{
		exact:    make(map[string]struct{}),
		prefixes: make([]string, 0, len(paths)),
	}

	seenPrefix := make(map[string]struct{})

	for _, path := range paths {
		path = strings.TrimSpace(path)
		path = strings.TrimPrefix(path, "/")
		if path == "" {
			continue
		}

		filter.exact[path] = struct{}{}

		pref := path
		if !strings.HasSuffix(pref, "/") {
			pref += "/"
		}
		if _, duplicate := seenPrefix[pref]; !duplicate {
			filter.prefixes = append(filter.prefixes, pref)
			seenPrefix[pref] = struct{}{}
		}
	}

	return filter
}

func (f *pathFilter) allow(path string) bool {
	// No filters -> allow all
	if len(f.exact) == 0 && len(f.prefixes) == 0 {
		return true
	}
	if _, ok := f.exact[path]; ok {
		return true
	}
	for _, prefix := range f.prefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}
