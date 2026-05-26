package main

import (
	"testing"

	"github.com/kardianos/service"
)

func TestIsHelpArg(t *testing.T) {
	for _, arg := range []string{"help", "--help", "-h", "/?"} {
		if !isHelpArg(arg) {
			t.Fatalf("isHelpArg(%q) = false, want true", arg)
		}
	}
	if isHelpArg("run") {
		t.Fatal("isHelpArg(\"run\") = true, want false")
	}
}

func TestServiceStatusText(t *testing.T) {
	tests := map[service.Status]string{
		service.StatusRunning: "running",
		service.StatusStopped: "stopped",
		service.StatusUnknown: "unknown (0)",
	}

	for status, want := range tests {
		if got := serviceStatusText(status); got != want {
			t.Fatalf("serviceStatusText(%d) = %q, want %q", status, got, want)
		}
	}
}

func TestParseServiceConfigPathArg(t *testing.T) {
	if got := parseServiceConfigPathArg([]string{"service", "--config", `C:\Agent\config.json`}); got != `C:\Agent\config.json` {
		t.Fatalf("parseServiceConfigPathArg() = %q", got)
	}
	if got := parseServiceConfigPathArg([]string{"service"}); got != "" {
		t.Fatalf("parseServiceConfigPathArg() = %q, want empty", got)
	}
}

func TestServiceArgumentsForConfigPath(t *testing.T) {
	args := serviceArgumentsForConfigPath(`C:\Agent\config.json`)
	want := []string{"service", "--config", `C:\Agent\config.json`}
	if len(args) != len(want) {
		t.Fatalf("len(args) = %d, want %d", len(args), len(want))
	}
	for i := range want {
		if args[i] != want[i] {
			t.Fatalf("args[%d] = %q, want %q", i, args[i], want[i])
		}
	}
}
