//go:build windows

package metrics

import (
	"sort"
	"strings"

	"github.com/yusufpapurcu/wmi"
)

type win32VideoController struct {
	Name           string
	DriverVersion  string
	AdapterRAM     uint32
	VideoProcessor string
	PNPDeviceID    string
}

func collectGPU() *GPUInfo {
	var controllers []win32VideoController
	if err := wmi.Query("SELECT Name, DriverVersion, AdapterRAM, VideoProcessor, PNPDeviceID FROM Win32_VideoController", &controllers); err != nil {
		return nil
	}

	devices := make([]GPUDevice, 0, len(controllers))
	for _, controller := range controllers {
		name := strings.TrimSpace(controller.Name)
		if name == "" {
			continue
		}
		devices = append(devices, GPUDevice{
			Name:           name,
			DriverVersion:  strings.TrimSpace(controller.DriverVersion),
			MemoryBytes:    uint64(controller.AdapterRAM),
			VideoProcessor: strings.TrimSpace(controller.VideoProcessor),
		})
	}
	if len(devices) == 0 {
		return nil
	}

	sort.SliceStable(devices, func(i, j int) bool {
		return gpuRank(devices[i].Name) > gpuRank(devices[j].Name)
	})

	primary := devices[0]
	return &GPUInfo{
		Name:           primary.Name,
		DriverVersion:  primary.DriverVersion,
		MemoryBytes:    primary.MemoryBytes,
		VideoProcessor: primary.VideoProcessor,
		Devices:        devices,
	}
}

func gpuRank(name string) int {
	name = strings.ToLower(name)
	switch {
	case strings.Contains(name, "nvidia"), strings.Contains(name, "geforce"), strings.Contains(name, "rtx"), strings.Contains(name, "gtx"):
		return 30
	case strings.Contains(name, "amd"), strings.Contains(name, "radeon"):
		return 20
	case strings.Contains(name, "intel"), strings.Contains(name, "uhd"), strings.Contains(name, "iris"):
		return 10
	default:
		return 0
	}
}
