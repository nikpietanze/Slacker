package main

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Release builds use the GUI subsystem so no console window appears at login.
// When started from a terminal we attach to it so CLI output still shows.

var (
	kernel32      = windows.NewLazySystemDLL("kernel32.dll")
	user32        = windows.NewLazySystemDLL("user32.dll")
	attachConsole = kernel32.NewProc("AttachConsole")
	getConsoleWnd = kernel32.NewProc("GetConsoleWindow")
	messageBox    = user32.NewProc("MessageBoxW")
	console       bool
)

func init() {
	if r, _, _ := getConsoleWnd.Call(); r != 0 {
		console = true
		return
	}
	const attachParentProcess = ^uintptr(0)
	if r, _, _ := attachConsole.Call(attachParentProcess); r != 0 {
		if f, err := os.OpenFile("CONOUT$", os.O_WRONLY, 0); err == nil {
			os.Stdout, os.Stderr = f, f
			console = true
		}
	}
}

func hasConsole() bool { return console }

func singleInstance() (func(), bool) {
	name, _ := windows.UTF16PtrFromString(`Local\SlackNotifyCompanion`)
	h, err := windows.CreateMutex(nil, false, name)
	if err == windows.ERROR_ALREADY_EXISTS {
		windows.CloseHandle(h)
		return func() {}, false
	}
	return func() { windows.CloseHandle(h) }, true
}

func msgBox(text string, flags uintptr) {
	t, _ := syscall.UTF16PtrFromString(text)
	c, _ := syscall.UTF16PtrFromString("SlackNotify Companion")
	messageBox.Call(0, uintptr(unsafe.Pointer(t)), uintptr(unsafe.Pointer(c)), flags)
}

// notifyStarted confirms a double-click launch, since there is no window.
func notifyStarted(dirs int) {
	if console {
		return
	}
	go msgBox(fmt.Sprintf("The companion is now running in the background and bridging to %d addon folder(s).\n\n"+
		"To start it automatically at login, run:\n  slacknotify-companion autostart on\n\n"+
		"To stop it, end \"slacknotify-companion\" in Task Manager.", dirs), 0x40 /* MB_ICONINFORMATION */)
}

func showError(err error) {
	if !console {
		msgBox(err.Error(), 0x10 /* MB_ICONERROR */)
	}
}
