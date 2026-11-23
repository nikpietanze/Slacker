//go:build !windows

package main

import (
	"os"
	"path/filepath"
	"syscall"
)

func hasConsole() bool {
	st, err := os.Stderr.Stat()
	return err == nil && st.Mode()&os.ModeCharDevice != 0
}

func singleInstance() (func(), bool) {
	p := filepath.Join(filepath.Dir(defaultLogPath()), "companion.lock")
	os.MkdirAll(filepath.Dir(p), 0o755)
	f, err := os.OpenFile(p, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return func() {}, true // can't lock; don't block the user
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		return func() {}, false
	}
	return func() { f.Close() }, true
}

func messageBox(string, string, bool) {} // Windows only; see desktop.go
