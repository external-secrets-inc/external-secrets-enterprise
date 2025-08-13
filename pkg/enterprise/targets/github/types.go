// /*
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

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
