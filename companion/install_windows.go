package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Installed copy: %LOCALAPPDATA%\Programs\Slacker\slacker-companion.exe
func installPath() string {
	return filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Slacker", "slacker-companion.exe")
}

func isInstalled(self string) bool {
	return strings.EqualFold(filepath.Clean(self), installPath())
}

func install(self string) (string, error) {
	target := installPath()
	return target, copyExecutable(self, target)
}

// terminateInstalled stops the installed copy (it has no window to close).
func terminateInstalled() {
	exec.Command("taskkill", "/F", "/IM", filepath.Base(installPath())).Run()
}

func startDetached(exe string, args ...string) error {
	return exec.Command(exe, args...).Start()
}
