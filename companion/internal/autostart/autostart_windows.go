package autostart

import (
	"strings"

	"golang.org/x/sys/windows/registry"
)

const runKey = `Software\Microsoft\Windows\CurrentVersion\Run`

// Enable starts exe with args at login (HKCU Run key).
func Enable(exe string, args []string) (string, error) {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, runKey, registry.SET_VALUE)
	if err != nil {
		return "", err
	}
	defer k.Close()
	parts := []string{`"` + exe + `"`}
	for _, a := range args {
		if strings.ContainsAny(a, " \t") {
			a = `"` + a + `"`
		}
		parts = append(parts, a)
	}
	return `HKCU\` + runKey + `\` + Name, k.SetStringValue(Name, strings.Join(parts, " "))
}

// Disable removes the login entry.
func Disable() error {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	if err := k.DeleteValue(Name); err != nil && err != registry.ErrNotExist {
		return err
	}
	return nil
}

// IsEnabled reports whether the login entry exists.
func IsEnabled() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	_, _, err = k.GetStringValue(Name)
	return err == nil
}
