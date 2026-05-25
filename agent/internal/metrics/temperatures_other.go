//go:build !windows

package metrics

func collectTemperatures() *TempInfo {
	return nil
}
