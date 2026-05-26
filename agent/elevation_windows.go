//go:build windows

package main

import (
	"github.com/lxn/win"
	"golang.org/x/sys/windows"
	"os"
	"strings"
	"syscall"
)

func ensureElevated() (bool, error) {
	elevated, err := isElevated()
	if err != nil {
		return false, err
	}
	if elevated {
		return false, nil
	}

	args := os.Args[1:]
	if len(args) == 0 {
		args = []string{"ui"}
	}
	ok := win.ShellExecute(0, utf16Ptr("runas"), utf16Ptr(os.Args[0]), utf16Ptr(joinCommandLine(args)), nil, win.SW_SHOWNORMAL)
	if !ok {
		return false, syscall.GetLastError()
	}
	return true, nil
}

func isElevated() (bool, error) {
	var token windows.Token
	if err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY, &token); err != nil {
		return false, err
	}
	defer token.Close()

	return token.IsElevated(), nil
}

func joinCommandLine(args []string) string {
	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		quoted = append(quoted, quoteArg(arg))
	}
	return strings.Join(quoted, " ")
}

func quoteArg(arg string) string {
	if arg == "" {
		return `""`
	}
	if !strings.ContainsAny(arg, " \t\"") {
		return arg
	}
	return `"` + strings.ReplaceAll(arg, `"`, `\"`) + `"`
}
