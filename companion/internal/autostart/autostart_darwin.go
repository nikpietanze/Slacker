package autostart

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
)

const label = "io.github.nikpietanze.slacker"

func plistPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", label+".plist")
}

func esc(s string) string {
	var b strings.Builder
	xml.EscapeText(&b, []byte(s))
	return b.String()
}

// Enable installs a LaunchAgent that starts exe with args at login.
func Enable(exe string, args []string) (string, error) {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
  <key>Label</key><string>` + label + `</string>
  <key>ProgramArguments</key><array>
    <string>` + esc(exe) + `</string>
`)
	for _, a := range args {
		b.WriteString("    <string>" + esc(a) + "</string>\n")
	}
	b.WriteString(`  </array>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><dict><key>SuccessfulExit</key><false/></dict>
  <key>ProcessType</key><string>Background</string>
</dict></plist>
`)
	p := plistPath()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(p, []byte(b.String()), 0o644); err != nil {
		return "", err
	}
	// Not loaded now: the running copy is already up, and launchd picks the
	// agent up at the next login.
	return p, nil
}

// IsEnabled reports whether the login item is installed.
func IsEnabled() bool {
	_, err := os.Stat(plistPath())
	return err == nil
}

// Disable unloads and removes the LaunchAgent.
func Disable() error {
	// Removing the plist stops it at the next login without stopping the
	// copy that's running now.
	p := plistPath()
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
