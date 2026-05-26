//go:build !windows

package metrics

func collectScreens() []ScreenInfo {
	return nil
}
