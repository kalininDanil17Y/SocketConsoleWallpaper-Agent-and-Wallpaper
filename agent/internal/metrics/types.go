package metrics

type AgentInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Status struct {
	Agent   AgentInfo    `json:"agent"`
	System  SystemInfo   `json:"system"`
	CPU     CPUInfo      `json:"cpu,omitempty"`
	Memory  MemoryInfo   `json:"memory,omitempty"`
	Network NetworkInfo  `json:"network,omitempty"`
	Disks   []DiskInfo   `json:"disks,omitempty"`
	Screens []ScreenInfo `json:"screens,omitempty"`
	GPU     *GPUInfo     `json:"gpu,omitempty"`
}

type SystemInfo struct {
	Hostname      string `json:"hostname"`
	OS            string `json:"os"`
	UptimeSeconds uint64 `json:"uptimeSeconds"`
}

type CPUInfo struct {
	Name         string  `json:"name"`
	UsagePercent float64 `json:"usagePercent"`
	Cores        int     `json:"cores"`
	Threads      int     `json:"threads"`
}

type MemoryInfo struct {
	UsedBytes    uint64  `json:"usedBytes"`
	TotalBytes   uint64  `json:"totalBytes"`
	UsagePercent float64 `json:"usagePercent"`
}

type NetworkInfo struct {
	SelectedInterface string `json:"selectedInterface"`
	IPv4              string `json:"ipv4"`
}

type DiskInfo struct {
	Name         string  `json:"name"`
	UsedBytes    uint64  `json:"usedBytes"`
	TotalBytes   uint64  `json:"totalBytes"`
	UsagePercent float64 `json:"usagePercent"`
}

type ScreenInfo struct {
	Name   string `json:"name"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type GPUInfo struct {
	Name           string      `json:"name"`
	DriverVersion  string      `json:"driverVersion,omitempty"`
	MemoryBytes    uint64      `json:"memoryBytes,omitempty"`
	VideoProcessor string      `json:"videoProcessor,omitempty"`
	Devices        []GPUDevice `json:"devices,omitempty"`
}

type GPUDevice struct {
	Name           string `json:"name"`
	DriverVersion  string `json:"driverVersion,omitempty"`
	MemoryBytes    uint64 `json:"memoryBytes,omitempty"`
	VideoProcessor string `json:"videoProcessor,omitempty"`
}

type InterfaceInfo struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"displayName,omitempty"`
	Up          bool     `json:"up"`
	Loopback    bool     `json:"loopback"`
	IPv4        []string `json:"ipv4"`
}

type DiskVolume struct {
	Name       string `json:"name"`
	Mountpoint string `json:"mountpoint"`
	Fstype     string `json:"fstype,omitempty"`
}
