package discovery

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if !config.Enabled {
		t.Error("Default config should be enabled")
	}
	if config.CacheTTL != DefaultCacheTTL {
		t.Error("Default cache TTL should match")
	}

	if len(config.Sources) > 0 {
		t.Error("Default config should not have any sources")
	}
}

func TestGetConfigFile(t *testing.T) {
	configDir := "/path/to/config"
	expected := filepath.Join(configDir, "autodiscover", "config.json")
	actual := GetConfigFile(configDir)
	if actual != expected {
		t.Errorf("Config file path should match. Got %v, want %v", actual, expected)
	}
}

func TestConfig_SaveAndLoadConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "javm-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configFile := filepath.Join(tempDir, "config.json")

	config := &Config{
		Enabled: true,
		Sources: map[string]bool{
			"system": true,
			"gradle": false,
		},
		CacheTTL: 2 * time.Hour,
	}

	err = config.SaveConfig(configFile)
	if err != nil {
		t.Errorf("SaveConfig should not return error: %v", err)
	}
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		t.Error("Config file should exist after SaveConfig")
	}

	loadedConfig, err := LoadConfig(configFile)
	if err != nil {
		t.Errorf("LoadConfig should not return error: %v", err)
	}
	if loadedConfig.Enabled != config.Enabled {
		t.Error("Enabled should match")
	}
	if loadedConfig.CacheTTL != config.CacheTTL {
		t.Error("CacheTTL should match")
	}
	if loadedConfig.Sources["system"] != config.Sources["system"] {
		t.Error("system should match")
	}
	if loadedConfig.Sources["gradle"] != config.Sources["gradle"] {
		t.Error("gradle should match")
	}

	nonExistentFile := filepath.Join(tempDir, "non-existent.json")
	loadedConfig, err = LoadConfig(nonExistentFile)
	if err != nil {
		t.Errorf("LoadConfig should not return error for non-existent file: %v", err)
	}
	if !loadedConfig.Enabled {
		t.Error("Enabled should be true for default config")
	}
	if loadedConfig.CacheTTL != DefaultCacheTTL {
		t.Error("CacheTTL should match default for non-existent file")
	}
	if len(loadedConfig.Sources) > 1 {
		t.Error("Default config should not have any sources")
	}

	nonExistentDir := filepath.Join(tempDir, "non-existent-dir", "config.json")
	err = config.SaveConfig(nonExistentDir)
	if err != nil {
		t.Errorf("SaveConfig should not return error for non-existent directory: %v", err)
	}
	if _, err := os.Stat(nonExistentDir); os.IsNotExist(err) {
		t.Error("Config file should exist after SaveConfig with non-existent directory")
	}
}

func TestConfig_IsSourceEnabled(t *testing.T) {
	config := &Config{
		Enabled: true,
		Sources: map[string]bool{
			"system": true,
			"gradle": false,
		},
	}
	if !config.IsSourceEnabled("system") {
		t.Error("system should be enabled")
	}
	if config.IsSourceEnabled("gradle") {
		t.Error("gradle should be disabled")
	}
	if !config.IsSourceEnabled("NonExistent") {
		t.Error("Non-existent source should be enabled")
	}

	config = &Config{
		Enabled: false,
		Sources: map[string]bool{
			"system": true,
			"gradle": true,
		},
	}
	if config.IsSourceEnabled("system") {
		t.Error("system should be disabled when autodiscovery is disabled")
	}
	if config.IsSourceEnabled("gradle") {
		t.Error("gradle should be disabled when autodiscovery is disabled")
	}
}
