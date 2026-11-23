package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"slacker/companion/internal/autostart"
)

// launch handles a double-click with no arguments: install to a permanent
// place, start at login, and get the tray icon up, with no terminal needed.
func launch(cfg config) error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	if p, err := filepath.EvalSymlinks(self); err == nil {
		self = p
	}
	installed := isInstalled(self)

	if release, ok := singleInstance(); ok {
		release()
	} else if installed {
		message("Slacker", "Slacker is already running. Look for the speech bubble in your "+trayName()+".", false)
		return nil
	} else if !stopRunning() {
		// A new download opened while an older copy runs: that's an update.
		message("Slacker", "Slacker is already running and couldn't be stopped for the update.\n\n"+
			"Quit Slacker from its "+trayName()+" icon, then open this file again.", true)
		return nil
	}

	if !installed {
		target, err := install(self)
		if err != nil {
			return fmt.Errorf("couldn't install Slacker: %w", err)
		}
		if err := enableAutostartFor(target, cfg); err != nil {
			return fmt.Errorf("couldn't set Slacker to start at login: %w", err)
		}
		return startDetached(target, "run", "--welcome")
	}

	// Already in place (for example dragged to Applications): make sure it
	// starts at login, then run here.
	welcome := !autostart.IsEnabled()
	if welcome {
		if err := enableAutostartFor(self, cfg); err != nil {
			return fmt.Errorf("couldn't set Slacker to start at login: %w", err)
		}
	}
	cfg.welcome = welcome
	return run(cfg, setupLog(cfg))
}

func enableAutostart(cfg config) error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	if p, err := filepath.EvalSymlinks(self); err == nil {
		self = p
	}
	return enableAutostartFor(self, cfg)
}

func enableAutostartFor(exe string, cfg config) error {
	args := []string{"run"}
	for _, d := range cfg.addonDirs {
		abs, _ := filepath.Abs(d)
		args = append(args, "--addon-dir", abs)
	}
	for _, d := range cfg.wowDirs {
		abs, _ := filepath.Abs(d)
		args = append(args, "--wow-dir", abs)
	}
	_, err := autostart.Enable(exe, args)
	return err
}

// welcomeText is shown once, right after the first install.
func welcomeText(w *watcher) string {
	var b strings.Builder
	b.WriteString("Slacker is running.\n\nLook for the speech bubble in your " + trayName() +
		". It starts by itself when you log in, so there's nothing else to do here.\n\n")
	if w.Status().Addons > 0 {
		b.WriteString("It found the Slacker addon in WoW, so you're all set.")
	} else {
		b.WriteString("Next, install the Slacker addon in WoW (for example from CurseForge). Slacker finds it automatically.")
	}
	switch runtime.GOOS {
	case "windows":
		b.WriteString("\n\nSlacker sees what Slack shows as Windows notifications, so keep those on in Slack's notification settings.")
	case "darwin":
		b.WriteString("\n\nSlacker reads the number on Slack's Dock icon, so keep \"Badge application icon\" on for Slack in System Settings > Notifications.")
	}
	return b.String()
}

// stopRunning asks the installed copy to quit (for an update) and waits up to
// five seconds for it to let go.
func stopRunning() bool {
	terminateInstalled()
	for i := 0; i < 50; i++ {
		if release, ok := singleInstance(); ok {
			release()
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

func copyExecutable(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp := dst + ".new"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, dst)
}
