//go:build windows

package metrics

import "syscall"

func collectScreens() []ScreenInfo {
	user32 := syscall.NewLazyDLL("user32.dll")
	getSystemMetrics := user32.NewProc("GetSystemMetrics")
	widthRaw, _, _ := getSystemMetrics.Call(0)
	heightRaw, _, _ := getSystemMetrics.Call(1)
	width := int(widthRaw)
	height := int(heightRaw)
	if width <= 0 || height <= 0 {
		return nil
	}
	return []ScreenInfo{{Name: "primary", Width: width, Height: height}}
}
