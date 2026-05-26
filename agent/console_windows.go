//go:build windows

package main

import "syscall"

var (
	kernel32Console = syscall.NewLazyDLL("kernel32.dll")
	freeConsole     = kernel32Console.NewProc("FreeConsole")
)

func hideConsoleForUI() {
	_, _, _ = freeConsole.Call()
}
