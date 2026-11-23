// Package autostart registers the companion to start when the user logs in.
// Everything is per-user; nothing needs administrator rights.
package autostart

import "strings"

// Name identifies the autostart entry on every platform.
const Name = "Slacker"

func quote(arg string) string {
	if arg != "" && !strings.ContainsAny(arg, " \t\"'\\$`") {
		return arg
	}
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "$", `\$`, "`", "\\`")
	return `"` + r.Replace(arg) + `"`
}
