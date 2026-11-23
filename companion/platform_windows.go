package main

import (
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
	messageBox_   = user32.NewProc("MessageBoxW")
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
	name, _ := windows.UTF16PtrFromString(`Local\SlackerCompanion`)
	h, err := windows.CreateMutex(nil, false, name)
	if err == windows.ERROR_ALREADY_EXISTS {
		windows.CloseHandle(h)
		return func() {}, false
	}
	return func() { windows.CloseHandle(h) }, true
}

func messageBox(title, text string, isError bool) {
	flags := uintptr(0x40) // MB_ICONINFORMATION
	if isError {
		flags = 0x10 // MB_ICONERROR
	}
	t, _ := syscall.UTF16PtrFromString(text)
	c, _ := syscall.UTF16PtrFromString(title)
	messageBox_.Call(0, uintptr(unsafe.Pointer(t)), uintptr(unsafe.Pointer(c)), flags|0x10000 /* MB_SETFOREGROUND */)
}
