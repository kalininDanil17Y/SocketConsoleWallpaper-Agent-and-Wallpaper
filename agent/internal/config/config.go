package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const (
	DefaultHost = "127.0.0.1"
	DefaultPort = 48771
)

type Config struct {
	Server  ServerConfig  `json:"server"`
	Metrics MetricsConfig `json:"metrics"`
	Network NetworkConfig `json:"network"`
	Disks   DisksConfig   `json:"disks"`
	Images  ImagesConfig  `json:"images"`
}

type ServerConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

type MetricsConfig struct {
	IntervalMs int  `json:"intervalMs"`
	CPU        bool `json:"cpu"`
	Memory     bool `json:"memory"`
	Disks      bool `json:"disks"`
	Network    bool `json:"network"`
	GPU        bool `json:"gpu"`
	Screens    bool `json:"screens"`
}

type NetworkConfig struct {
	InterfaceName string `json:"interfaceName"`
	PreferIPv4    bool   `json:"preferIPv4"`
}

type DisksConfig struct {
	Include []string `json:"include"`
}

type ImagesConfig struct {
	Directory          string `json:"directory"`
	ChangeEverySeconds int    `json:"changeEverySeconds"`
	ASCIIWidth         int    `json:"asciiWidth"`
	ASCIIHeight        int    `json:"asciiHeight"`
	Charset            string `json:"charset"`
	PaletteSize        int    `json:"paletteSize"`
}

func Load(path string) (*Config, string, error) {
	resolved, err := filepath.Abs(path)
	if err != nil {
		return nil, "", err
	}

	cfg := Default()
	if _, err := os.Stat(resolved); errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(filepath.Dir(resolved), 0o755); err != nil {
			return nil, "", err
		}
		if err := Save(resolved, cfg); err != nil {
			return nil, "", err
		}
		return cfg, resolved, nil
	}

	data, err := os.ReadFile(resolved)
	if err != nil {
		return nil, "", err
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, "", err
	}
	cfg.Normalize()
	return cfg, resolved, nil
}

func Save(path string, cfg *Config) error {
	cfg.Normalize()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func Default() *Config {
	imageDir := filepath.Join(userHomeDir(), "Pictures", "WallpaperAscii")
	return &Config{
		Server: ServerConfig{
			Host: DefaultHost,
			Port: DefaultPort,
		},
		Metrics: MetricsConfig{
			IntervalMs: 5000,
			CPU:        true,
			Memory:     true,
			Disks:      true,
			Network:    true,
			GPU:        true,
			Screens:    true,
		},
		Network: NetworkConfig{
			InterfaceName: "",
			PreferIPv4:    true,
		},
		Disks: DisksConfig{
			Include: []string{"C:"},
		},
		Images: ImagesConfig{
			Directory:          imageDir,
			ChangeEverySeconds: 5,
			ASCIIWidth:         90,
			ASCIIHeight:        45,
			Charset:            " .:-=+*#%@",
			PaletteSize:        128,
		},
	}
}

func (c *Config) Normalize() {
	if c.Server.Host == "" {
		c.Server.Host = DefaultHost
	}
	if c.Server.Host != DefaultHost {
		c.Server.Host = DefaultHost
	}
	if c.Server.Port == 0 {
		c.Server.Port = DefaultPort
	}
	if c.Metrics.IntervalMs < 1000 {
		c.Metrics.IntervalMs = 5000
	}
	if c.Images.ChangeEverySeconds <= 0 {
		c.Images.ChangeEverySeconds = 5
	}
	if c.Images.ASCIIWidth <= 0 {
		c.Images.ASCIIWidth = 90
	}
	if c.Images.ASCIIHeight <= 0 {
		c.Images.ASCIIHeight = 45
	}
	if c.Images.Charset == "" {
		c.Images.Charset = " .:-=+*#%@"
	}
	if c.Images.PaletteSize <= 0 {
		c.Images.PaletteSize = 128
	}
	if c.Images.Directory == "" {
		c.Images.Directory = filepath.Join(userHomeDir(), "Pictures", "WallpaperAscii")
	}
}

func DevConfigPath() string {
	if path := os.Getenv("SOCKET_CONSOLE_AGENT_CONFIG"); path != "" {
		return path
	}
	return filepath.Join(".", "config.json")
}

func ServiceConfigPath() string {
	if path := os.Getenv("SOCKET_CONSOLE_AGENT_CONFIG"); path != "" {
		return path
	}
	if programData := os.Getenv("ProgramData"); programData != "" {
		return filepath.Join(programData, "SocketConsoleAgent", "config.json")
	}
	return filepath.Join(userConfigDir(), "SocketConsoleAgent", "config.json")
}

func userHomeDir() string {
	if dir, err := os.UserHomeDir(); err == nil && dir != "" {
		return dir
	}
	return "."
}

func userConfigDir() string {
	if dir, err := os.UserConfigDir(); err == nil && dir != "" {
		return dir
	}
	return "."
}
