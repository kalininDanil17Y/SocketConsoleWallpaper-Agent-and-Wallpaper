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
