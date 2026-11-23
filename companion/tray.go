package main

import (
	"context"
	_ "embed"
	"fmt"
	"runtime"

	"fyne.io/systray"

	"slacker/companion/internal/autostart"
)

const (
	projectURL = "https://github.com/nikpietanze/Slacker"
	addonURL   = projectURL + "#install"
	helpURL    = projectURL + "/blob/main/docs/TROUBLESHOOTING.md"
)

// Tray icons, drawn by cmd/genart (make art).
var (
	//go:embed assets/tray.ico
	trayICO []byte
	//go:embed assets/tray-unread.ico
	trayUnreadICO []byte
	//go:embed assets/tray.png
	trayPNG []byte
	//go:embed assets/tray-unread.png
	trayUnreadPNG []byte
	//go:embed assets/tray-template.png
	trayTemplatePNG []byte
)

func setTrayIcon(unread bool) {
	switch runtime.GOOS {
	case "windows":
		if unread {
			systray.SetIcon(trayUnreadICO)
		} else {
			systray.SetIcon(trayICO)
		}
	case "darwin":
		// A template icon follows the menu bar's light/dark look; the count
		// shows as text beside it.
		systray.SetTemplateIcon(trayTemplatePNG, trayPNG)
	default:
		if unread {
			systray.SetIcon(trayUnreadPNG)
		} else {
			systray.SetIcon(trayPNG)
		}
	}
}

func trayName() string {
	if runtime.GOOS == "darwin" {
		return "menu bar"
	}
	return "system tray"
}

// runTray shows the tray/menu bar icon and blocks until the user quits.
// It must run on the main goroutine.
func runTray(w *watcher, cancel context.CancelFunc) {
	systray.Run(func() { trayReady(w) }, cancel)
}

func trayReady(w *watcher) {
	setTrayIcon(false)
	systray.SetTooltip("Slacker")

	slackItem := systray.AddMenuItem("Slack: starting…", "")
	slackItem.Disable()
	wowItem := systray.AddMenuItem("WoW: looking for the Slacker addon…", "")
	systray.AddSeparator()
	loginItem := systray.AddMenuItemCheckbox("Start when I log in", "", autostart.IsEnabled())
	helpItem := systray.AddMenuItem("Help", "")
	quitItem := systray.AddMenuItem("Quit Slacker", "")

	lastUnread := false
	refresh := func() {
		st := w.Status()
		var slack, tip string
		switch {
		case st.Err != "":
			slack, tip = "Can't read Slack notifications (see Help)", "Slacker: can't read Slack notifications"
		case st.Count < 0:
			slack, tip = "Slack: starting…", "Slacker"
		case st.Count == 0:
			slack, tip = "Slack: no notifications", "Slacker: no Slack notifications"
		default:
			n := fmt.Sprint(st.Count)
			if st.Count > 99 {
				n = "99+"
			}
			slack = fmt.Sprintf("Slack: %s notification%s", n, plural(st.Count))
			tip = "Slacker: " + slack
		}
		slackItem.SetTitle(slack)
		systray.SetTooltip(tip)

		if st.Addons > 0 {
			wowItem.SetTitle("WoW: connected to the Slacker addon")
			wowItem.Disable()
		} else {
			wowItem.SetTitle("WoW: Slacker addon not found. Get it…")
			wowItem.Enable()
		}

		unread := st.Count > 0
		if runtime.GOOS == "darwin" {
			if unread {
				systray.SetTitle(fmt.Sprint(min(st.Count, 99)))
			} else {
				systray.SetTitle("")
			}
		} else if unread != lastUnread {
			setTrayIcon(unread)
		}
		lastUnread = unread
	}
	w.mu.Lock()
	w.onChange = refresh
	w.mu.Unlock()
	refresh()

	go func() {
		for {
			select {
			case <-wowItem.ClickedCh:
				openURL(addonURL)
			case <-helpItem.ClickedCh:
				openURL(helpURL)
			case <-loginItem.ClickedCh:
				if loginItem.Checked() {
					if err := autostart.Disable(); err == nil {
						loginItem.Uncheck()
					}
				} else if err := enableAutostart(w.cfg); err == nil {
					loginItem.Check()
				} else {
					w.log.Printf("autostart: %v", err)
				}
			case <-quitItem.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
