//go:build !windows

package metrics

func collectGPU() *GPUInfo {
	return nil
}
