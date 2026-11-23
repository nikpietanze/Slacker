package bridge

import (
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

func defaultWoWRoots() []string {
	var roots []string
	// The Battle.net installer records the game folder (it points at a flavor
	// folder such as ...\World of Warcraft\_retail_\, so take its parent).
	for _, key := range []string{
		`SOFTWARE\WOW6432Node\Blizzard Entertainment\World of Warcraft`,
		`SOFTWARE\Blizzard Entertainment\World of Warcraft`,
	} {
		k, err := registry.OpenKey(registry.LOCAL_MACHINE, key, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		if p, _, err := k.GetStringValue("InstallPath"); err == nil && p != "" {
			p = filepath.Clean(p)
			if strings.HasPrefix(filepath.Base(p), "_") {
				p = filepath.Dir(p)
			}
			roots = append(roots, p)
		}
		k.Close()
	}
	var patterns []string
	for _, pf := range []string{os.Getenv("ProgramFiles(x86)"), os.Getenv("ProgramFiles")} {
		if pf != "" {
			patterns = append(patterns, filepath.Join(pf, "World of Warcraft"))
		}
	}
	for _, drive := range []string{"C", "D", "E", "F"} {
		patterns = append(patterns,
			drive+`:\World of Warcraft`,
			drive+`:\Games\World of Warcraft`,
			drive+`:\Program Files (x86)\World of Warcraft`,
			drive+`:\Program Files\World of Warcraft`,
		)
	}
	return append(roots, expandGlobs(patterns)...)
}
