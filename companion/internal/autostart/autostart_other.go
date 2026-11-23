//go:build !windows && !darwin

package autostart

import (
	"os"
	"path/filepath"
	"strings"
)

func desktopPath() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "autostart", "slacknotify-companion.desktop")
}

// Enable writes an XDG autostart entry that starts exe with args at login.
func Enable(exe string, args []string) (string, error) {
	cmd := []string{quote(exe)}
	for _, a := range args {
		cmd = append(cmd, quote(a))
	}
	content := "[Desktop Entry]\nType=Application\nName=Slack Notify Companion\n" +
		"Comment=Mirrors Slack notification count into the WoW SlackNotify addon\n" +
		"Exec=" + strings.Join(cmd, " ") + "\nTerminal=false\nNoDisplay=true\nX-GNOME-Autostart-enabled=true\n"
	p := desktopPath()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return "", err
	}
	return p, os.WriteFile(p, []byte(content), 0o644)
}

// Disable removes the autostart entry.
func Disable() error {
	if err := os.Remove(desktopPath()); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
