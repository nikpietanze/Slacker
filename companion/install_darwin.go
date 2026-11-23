package main

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// On macOS the companion ships as "Slacker Companion.app". Opened from
// Downloads (or wherever the zip unpacked), it copies itself to
// ~/Applications so the login item keeps working after Downloads is cleaned.

const bundleName = "Slacker Companion.app"

// bundleOf returns the .app containing exe, if any.
func bundleOf(exe string) (string, bool) {
	macOS := filepath.Dir(exe)
	contents := filepath.Dir(macOS)
	bundle := filepath.Dir(contents)
	if filepath.Base(macOS) == "MacOS" && filepath.Base(contents) == "Contents" && strings.HasSuffix(bundle, ".app") {
		return bundle, true
	}
	return "", false
}

func isInstalled(self string) bool {
	bundle, ok := bundleOf(self)
	if !ok {
		return true // a bare binary (developer build): run it where it is
	}
	home, _ := os.UserHomeDir()
	dir := filepath.Dir(bundle)
	return dir == "/Applications" || dir == filepath.Join(home, "Applications")
}

func install(self string) (string, error) {
	bundle, _ := bundleOf(self)
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	target := filepath.Join(home, "Applications", bundleName)
	tmp := target + ".new"
	os.RemoveAll(tmp)
	if err := copyTree(bundle, tmp); err != nil {
		os.RemoveAll(tmp)
		return "", err
	}
	os.RemoveAll(target)
	if err := os.Rename(tmp, target); err != nil {
		return "", err
	}
	return filepath.Join(target, "Contents", "MacOS", filepath.Base(self)), nil
}

func copyTree(src, dst string) error {
	return filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, rel)
		switch {
		case info.IsDir():
			return os.MkdirAll(target, info.Mode().Perm()|0o700)
		case info.Mode()&os.ModeSymlink != 0:
			link, err := os.Readlink(p)
			if err != nil {
				return err
			}
			return os.Symlink(link, target)
		default:
			in, err := os.Open(p)
			if err != nil {
				return err
			}
			defer in.Close()
			out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode().Perm())
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, in); err != nil {
				out.Close()
				return err
			}
			return out.Close()
		}
	})
}

// terminateInstalled asks any installed copy to quit; it writes "offline"
// to the addon on the way out.
// Only copies in an Applications folder: the new copy that's doing the
// update runs from somewhere else (usually Downloads).
func terminateInstalled() {
	home, _ := os.UserHomeDir()
	for _, dir := range []string{filepath.Join(home, "Applications"), "/Applications"} {
		exec.Command("pkill", "-TERM", "-f", filepath.Join(dir, bundleName, "Contents", "MacOS")+"/").Run()
	}
}

func startDetached(exe string, args ...string) error {
	if bundle, ok := bundleOf(exe); ok {
		return exec.Command("open", append([]string{"-n", bundle, "--args"}, args...)...).Run()
	}
	return exec.Command(exe, args...).Start()
}
