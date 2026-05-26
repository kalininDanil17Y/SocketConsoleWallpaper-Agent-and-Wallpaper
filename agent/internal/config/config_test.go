package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeAppliesSafeDefaults(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
		},
		Metrics: MetricsConfig{
			IntervalMs: 250,
		},
		Images: ImagesConfig{
			ChangeEverySeconds: -1,
			ASCIIWidth:         -10,
			ASCIIHeight:        0,
			PaletteSize:        -5,
		},
	}

	cfg.Normalize()

	if cfg.Server.Host != DefaultHost {
		t.Fatalf("Server.Host = %q, want %q", cfg.Server.Host, DefaultHost)
	}
	if cfg.Server.Port != DefaultPort {
		t.Fatalf("Server.Port = %d, want %d", cfg.Server.Port, DefaultPort)
	}
	if cfg.Metrics.IntervalMs != 5000 {
		t.Fatalf("Metrics.IntervalMs = %d, want 5000", cfg.Metrics.IntervalMs)
	}
	if cfg.Images.ChangeEverySeconds != 5 {
		t.Fatalf("Images.ChangeEverySeconds = %d, want 5", cfg.Images.ChangeEverySeconds)
	}
	if cfg.Images.ASCIIWidth != 90 || cfg.Images.ASCIIHeight != 45 {
		t.Fatalf("ASCII size = %dx%d, want 90x45", cfg.Images.ASCIIWidth, cfg.Images.ASCIIHeight)
	}
	if cfg.Images.Charset != " .:-=+*#%@" {
		t.Fatalf("Images.Charset = %q, want default charset", cfg.Images.Charset)
	}
	if cfg.Images.PaletteSize != 128 {
		t.Fatalf("Images.PaletteSize = %d, want 128", cfg.Images.PaletteSize)
	}
	if cfg.Images.Directory == "" {
		t.Fatal("Images.Directory is empty after Normalize")
	}
}

func TestLoadCreatesDefaultConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.json")

	cfg, resolved, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if resolved == "" {
		t.Fatal("Load() returned empty resolved path")
	}
	if cfg.Server.Host != DefaultHost || cfg.Server.Port != DefaultPort {
		t.Fatalf("loaded server config = %+v, want default localhost config", cfg.Server)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("default config was not written: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var saved Config
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatalf("saved config is invalid JSON: %v", err)
	}
	if saved.Images.Directory == "" {
		t.Fatal("saved config has empty image directory")
	}
}

func TestSaveNormalizesBeforeWriting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	cfg := &Config{
		Server: ServerConfig{Host: "192.168.1.10"},
		Images: ImagesConfig{ASCIIWidth: 1, ASCIIHeight: 2, Charset: "01", PaletteSize: 2},
	}

	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var saved Config
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatalf("saved config is invalid JSON: %v", err)
	}
	if saved.Server.Host != DefaultHost {
		t.Fatalf("saved Server.Host = %q, want %q", saved.Server.Host, DefaultHost)
	}
	if saved.Server.Port != DefaultPort {
		t.Fatalf("saved Server.Port = %d, want %d", saved.Server.Port, DefaultPort)
	}
}
