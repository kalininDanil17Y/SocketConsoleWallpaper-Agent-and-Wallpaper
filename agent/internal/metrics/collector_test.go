package metrics

import (
	"math"
	"net"
	"testing"

	"github.com/shirou/gopsutil/v4/host"
)

func TestNormalizeDiskName(t *testing.T) {
	tests := map[string]string{
		`C:\`:     "C:",
		"c:":      "C:",
		" D: ":    "D:",
		"/mnt/d":  "/mnt/d:",
		"":        "",
		`E:\Data`: "E:",
	}

	for input, want := range tests {
		if got := normalizeDiskName(input); got != want {
			t.Fatalf("normalizeDiskName(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestAddrIP(t *testing.T) {
	ipNet := &net.IPNet{IP: net.ParseIP("192.168.1.10"), Mask: net.CIDRMask(24, 32)}
	if got := addrIP(ipNet); !got.Equal(net.ParseIP("192.168.1.10")) {
		t.Fatalf("addrIP(IPNet) = %v, want 192.168.1.10", got)
	}

	ipAddr := &net.IPAddr{IP: net.ParseIP("10.0.0.5")}
	if got := addrIP(ipAddr); !got.Equal(net.ParseIP("10.0.0.5")) {
		t.Fatalf("addrIP(IPAddr) = %v, want 10.0.0.5", got)
	}
}

func TestRound1HandlesInvalidValues(t *testing.T) {
	if got := round1(12.34); got != 12.3 {
		t.Fatalf("round1(12.34) = %.1f, want 12.3", got)
	}
	if got := round1(12.35); got != 12.4 {
		t.Fatalf("round1(12.35) = %.1f, want 12.4", got)
	}
	if got := round1(math.Inf(1)); got != 0 {
		t.Fatalf("round1(+Inf) = %.1f, want 0", got)
	}
}

func TestOSNameTrimsAndFallsBack(t *testing.T) {
	withPlatform := osName(&host.InfoStat{
		Platform:        " windows ",
		PlatformVersion: " 11 ",
		OS:              " ignored ",
	})
	if withPlatform != "windows 11" {
		t.Fatalf("osName(with platform) = %q, want %q", withPlatform, "windows 11")
	}

	fallback := osName(&host.InfoStat{OS: " linux "})
	if fallback != "linux" {
		t.Fatalf("osName(fallback) = %q, want %q", fallback, "linux")
	}
}
