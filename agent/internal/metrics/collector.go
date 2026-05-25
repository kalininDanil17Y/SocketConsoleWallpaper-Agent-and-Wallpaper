package metrics

import (
	"fmt"
	"math"
	"net"
	"runtime"
	"sort"
	"strings"
	"syscall"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"

	"socket-console-agent/internal/config"
)

const (
	AgentName    = "Socket Console Agent"
	AgentVersion = "0.1.0"
)

type Collector struct{}

func NewCollector() *Collector {
	return &Collector{}
}

func (c *Collector) Collect(cfg *config.Config) (Status, error) {
	status := Status{
		Agent: AgentInfo{
			Name:    AgentName,
			Version: AgentVersion,
		},
	}

	if info, err := host.Info(); err == nil {
		status.System = SystemInfo{
			Hostname:      info.Hostname,
			OS:            osName(info),
			UptimeSeconds: info.Uptime,
		}
	} else {
		return status, err
	}

	if cfg.Metrics.CPU {
		status.CPU = collectCPU()
	}
	if cfg.Metrics.Memory {
		status.Memory = collectMemory()
	}
	if cfg.Metrics.Network {
		status.Network = collectNetwork(cfg.Network.InterfaceName)
	}
	if cfg.Metrics.Disks {
		status.Disks = collectDisks(cfg.Disks.Include)
	}
	if cfg.Metrics.Screens {
		status.Screens = collectScreens()
	}
	if cfg.Metrics.GPU {
		status.GPU = &OptionalInfo{Enabled: false, Reason: "GPU collection is not implemented yet"}
	}

	return status, nil
}

func Interfaces() ([]InterfaceInfo, error) {
	nics, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	result := make([]InterfaceInfo, 0, len(nics))
	for _, nic := range nics {
		addrs, _ := nic.Addrs()
		info := InterfaceInfo{
			Name:     nic.Name,
			Up:       nic.Flags&net.FlagUp != 0,
			Loopback: nic.Flags&net.FlagLoopback != 0,
		}
		for _, addr := range addrs {
			ip := addrIP(addr)
			if ip4 := ip.To4(); ip4 != nil {
				info.IPv4 = append(info.IPv4, ip4.String())
			}
		}
		result = append(result, info)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Up != result[j].Up {
			return result[i].Up
		}
		return result[i].Name < result[j].Name
	})
	return result, nil
}

func Disks() ([]DiskVolume, error) {
	partitions, err := disk.Partitions(false)
	if err != nil {
		return nil, err
	}

	volumes := make([]DiskVolume, 0, len(partitions))
	seen := map[string]bool{}
	for _, partition := range partitions {
		name := normalizeDiskName(partition.Mountpoint)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		volumes = append(volumes, DiskVolume{
			Name:       name,
			Mountpoint: partition.Mountpoint,
			Fstype:     partition.Fstype,
		})
	}
	sort.Slice(volumes, func(i, j int) bool {
		return volumes[i].Name < volumes[j].Name
	})
	return volumes, nil
}

func collectCPU() CPUInfo {
	out := CPUInfo{}
	if infos, err := cpu.Info(); err == nil && len(infos) > 0 {
		out.Name = infos[0].ModelName
	}
	if physical, err := cpu.Counts(false); err == nil {
		out.Cores = physical
	}
	if logical, err := cpu.Counts(true); err == nil {
		out.Threads = logical
	}
	if percents, err := cpu.Percent(0, false); err == nil && len(percents) > 0 {
		out.UsagePercent = round1(percents[0])
	}
	return out
}

func collectMemory() MemoryInfo {
	if vm, err := mem.VirtualMemory(); err == nil {
		return MemoryInfo{
			UsedBytes:    vm.Used,
			TotalBytes:   vm.Total,
			UsagePercent: round1(vm.UsedPercent),
		}
	}
	return MemoryInfo{}
}

func collectNetwork(preferred string) NetworkInfo {
	interfaces, err := Interfaces()
	if err != nil {
		return NetworkInfo{}
	}

	for _, nic := range interfaces {
		if preferred != "" && !strings.EqualFold(nic.Name, preferred) {
			continue
		}
		if len(nic.IPv4) > 0 {
			return NetworkInfo{SelectedInterface: nic.Name, IPv4: nic.IPv4[0]}
		}
	}
	for _, nic := range interfaces {
		if nic.Up && !nic.Loopback && len(nic.IPv4) > 0 {
			return NetworkInfo{SelectedInterface: nic.Name, IPv4: nic.IPv4[0]}
		}
	}
	return NetworkInfo{}
}

func collectDisks(include []string) []DiskInfo {
	volumes, err := Disks()
	if err != nil {
		return nil
	}
	includeSet := make(map[string]bool, len(include))
	for _, name := range include {
		includeSet[normalizeDiskName(name)] = true
	}

	result := make([]DiskInfo, 0, len(volumes))
	for _, volume := range volumes {
		if len(includeSet) > 0 && !includeSet[volume.Name] {
			continue
		}
		usage, err := disk.Usage(volume.Mountpoint)
		if err != nil {
			continue
		}
		result = append(result, DiskInfo{
			Name:         volume.Name,
			UsedBytes:    usage.Used,
			TotalBytes:   usage.Total,
			UsagePercent: round1(usage.UsedPercent),
		})
	}
	return result
}

func collectScreens() []ScreenInfo {
	if runtime.GOOS != "windows" {
		return nil
	}
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

func osName(info *host.InfoStat) string {
	parts := []string{}
	if info.Platform != "" {
		parts = append(parts, info.Platform)
	}
	if info.PlatformVersion != "" {
		parts = append(parts, info.PlatformVersion)
	}
	if len(parts) == 0 {
		return info.OS
	}
	return strings.Join(parts, " ")
}

func normalizeDiskName(name string) string {
	name = strings.TrimSpace(strings.ReplaceAll(name, "\\", ""))
	if name == "" {
		return ""
	}
	if len(name) >= 2 && name[1] == ':' {
		return strings.ToUpper(name[:2])
	}
	return strings.TrimRight(name, ":") + ":"
}

func addrIP(addr net.Addr) net.IP {
	switch v := addr.(type) {
	case *net.IPNet:
		return v.IP
	case *net.IPAddr:
		return v.IP
	default:
		ip, _, err := net.ParseCIDR(addr.String())
		if err != nil {
			return nil
		}
		return ip
	}
}

func round1(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return math.Round(v*10) / 10
}

func (s Status) String() string {
	return fmt.Sprintf("%s %s, CPU %.1f%%, RAM %.1f%%", s.Agent.Name, s.Agent.Version, s.CPU.UsagePercent, s.Memory.UsagePercent)
}
