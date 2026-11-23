package main

import (
	"os/exec"
	"runtime"
	"strings"
)

// openURL opens a web page in the default browser.
func openURL(url string) {
	switch runtime.GOOS {
	case "windows":
		exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		exec.Command("open", url).Start()
	default:
		exec.Command("xdg-open", url).Start()
	}
}

// message shows a native message box. Windows uses MessageBox (see
// platform_windows.go); macOS uses an AppleScript dialog; Linux tries zenity,
// then kdialog, then a desktop notification.
func message(title, text string, isError bool) {
	switch runtime.GOOS {
	case "windows":
		messageBox(title, text, isError)
	case "darwin":
		icon := "note"
		if isError {
			icon = "caution"
		}
		script := `display dialog "` + appleQuote(text) + `" with title "` + appleQuote(title) +
			`" buttons {"OK"} default button "OK" with icon ` + icon
		exec.Command("osascript", "-e", script).Run()
	default:
		if p, err := exec.LookPath("zenity"); err == nil {
			kind := "--info"
			if isError {
				kind = "--error"
			}
			exec.Command(p, kind, "--title", title, "--text", text, "--no-wrap").Run()
		} else if p, err := exec.LookPath("kdialog"); err == nil {
			kind := "--msgbox"
			if isError {
				kind = "--error"
			}
			exec.Command(p, "--title", title, kind, text).Run()
		} else if p, err := exec.LookPath("notify-send"); err == nil {
			exec.Command(p, "-a", "Slacker", title, text).Run()
		}
	}
}

func appleQuote(s string) string {
	return strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s)
}
