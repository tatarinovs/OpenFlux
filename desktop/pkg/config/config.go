package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type Config struct {
	DocURLs        string `json:"doc_urls"`
	SecretKey      string `json:"secret_key"`
	SocksPort      int    `json:"socks_port"`
	Mode           string `json:"mode"`            // "wintun", "sysproxy", "socks"
	Bypass         string `json:"bypass"`          // custom bypass addresses
	AutoStart      bool   `json:"auto_start"`      // start with Windows
	StartMinimized bool   `json:"start_minimized"` // start minimized to tray
	CloseToTray    bool   `json:"close_to_tray"`   // minimize to tray on window close
	Debug          bool   `json:"debug"`
}

var (
	cfgMu      sync.RWMutex
	currentCfg Config
)

func DefaultConfig() Config {
	return Config{
		DocURLs:        "",
		SecretKey:      "",
		SocksPort:      1080,
		Mode:           "sysproxy",
		Bypass:         "<local>;localhost;127.*;192.168.*;10.*",
		AutoStart:      false,
		StartMinimized: false,
		CloseToTray:    false,
		Debug:          false,
	}
}

func GetConfigDir() string {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		appData = "."
	}
	dir := filepath.Join(appData, "OpenFlux")
	_ = os.MkdirAll(dir, 0755)
	return dir
}

func GetConfigFile() string {
	return filepath.Join(GetConfigDir(), "config.json")
}

func Load() Config {
	cfgMu.Lock()
	defer cfgMu.Unlock()

	cfg := DefaultConfig()
	data, err := os.ReadFile(GetConfigFile())
	if err == nil {
		_ = json.Unmarshal(data, &cfg)
	}
	if cfg.SocksPort <= 0 {
		cfg.SocksPort = 1080
	}
	if cfg.Mode == "" {
		cfg.Mode = "sysproxy"
	}
	if cfg.Bypass == "" {
		cfg.Bypass = "<local>;localhost;127.*;192.168.*;10.*"
	}
	currentCfg = cfg
	return cfg
}

func Save(cfg Config) error {
	cfgMu.Lock()
	defer cfgMu.Unlock()

	if cfg.SocksPort <= 0 {
		cfg.SocksPort = 1080
	}
	currentCfg = cfg
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(GetConfigFile(), data, 0644)
}

func Get() Config {
	cfgMu.RLock()
	defer cfgMu.RUnlock()
	return currentCfg
}
