//go:build !windows

package bridge

import (
	"os"
	"path/filepath"
	"runtime"
)

func defaultWoWRoots() []string {
	home, _ := os.UserHomeDir()
	var patterns []string
	if runtime.GOOS == "darwin" {
		patterns = []string{
			"/Applications/World of Warcraft",
			filepath.Join(home, "Applications", "World of Warcraft"),
		}
	} else {
		// Linux: WoW runs under Wine/Proton. Cover the common prefixes.
		wow := filepath.Join("drive_c", "Program Files*", "World of Warcraft")
		patterns = []string{
			filepath.Join(home, ".wine", wow),
			filepath.Join(home, "Games", "*", wow),
			filepath.Join(home, "Games", "*", "*", wow),
			filepath.Join(home, ".local", "share", "lutris", "runners", "wine", "*", wow),
			filepath.Join(home, ".steam", "steam", "steamapps", "compatdata", "*", "pfx", wow),
			filepath.Join(home, ".local", "share", "Steam", "steamapps", "compatdata", "*", "pfx", wow),
			filepath.Join(home, ".local", "share", "bottles", "bottles", "*", wow),
			filepath.Join(home, ".var", "app", "com.usebottles.bottles", "data", "bottles", "bottles", "*", wow),
		}
	}
	return expandGlobs(patterns)
}
