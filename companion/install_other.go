//go:build !windows && !darwin

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// Installed copy: ~/.local/bin/slacker-companion
func installPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "bin", "slacker-companion")
}

func isInstalled(self string) bool {
	return filepath.Clean(self) == installPath()
}

func install(self string) (string, error) {
	target := installPath()
	return target, copyExecutable(self, target)
}

func startDetached(exe string, args ...string) error {
	cmd := exec.Command(exe, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	return cmd.Start()
}
