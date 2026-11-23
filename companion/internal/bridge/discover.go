package bridge

import (
	"os"
	"path/filepath"
	"sort"
)

// Discover returns every installed Slacker addon folder it can find.
//
// explicitAddonDirs are used as-is. wowRoots are World of Warcraft install
// folders (the one that contains _forever_, _retail_, ...). When both are
// empty, platform default locations are searched.
func Discover(explicitAddonDirs, wowRoots []string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(p string) {
		p = filepath.Clean(p)
		if seen[p] {
			return
		}
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			seen[p] = true
			out = append(out, p)
		}
	}
	for _, d := range explicitAddonDirs {
		add(d)
	}
	roots := wowRoots
	if len(explicitAddonDirs) == 0 && len(roots) == 0 {
		roots = defaultWoWRoots()
	}
	for _, root := range roots {
		// Every game flavor folder: _forever_, _retail_, _classic_, _beta_ ...
		matches, _ := filepath.Glob(filepath.Join(root, "_*_", "Interface", "AddOns", AddonName))
		sort.Strings(matches)
		for _, m := range matches {
			add(m)
		}
	}
	return out
}

// expandGlobs resolves glob patterns to existing directories.
func expandGlobs(patterns []string) []string {
	var out []string
	for _, p := range patterns {
		m, _ := filepath.Glob(p)
		out = append(out, m...)
	}
	return out
}
