package discovery

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/felipebz/javm/internal/state"
)

type Config struct {
	Enabled  bool            `json:"enabled"`
	Sources  map[string]bool `json:"sources"`
	CacheTTL time.Duration   `json:"cache_ttl"`
}

// DefaultConfig returns the default configuration for autodiscovery
func DefaultConfig() *Config {
	return &Config{
		Enabled:  true,
		CacheTTL: DefaultCacheTTL,
	}
}

func GetConfigFile(configDir string) string {
	return filepath.Join(configDir, "autodiscover", "config.json")
}

func LoadConfig(configFile string) (*Config, error) {
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		// Return the default config if the file doesn't exist
		return DefaultConfig(), nil
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func (c *Config) SaveConfig(configFile string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return state.WithFileLock(configFile, func() error {
		return state.AtomicWriteFile(configFile, data, 0o600)
	})
}

func (c *Config) IsSourceEnabled(source string) bool {
	if !c.Enabled {
		return false
	}
	enabled, ok := c.Sources[source]
	return !ok || enabled
}

func (c *Config) fingerprint() (string, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("encode discovery configuration: %w", err)
	}
	return fmt.Sprintf("%x", sha256.Sum256(data)), nil
}
