package discovery

import (
	"os"
	"path/filepath"
	"time"
)

const DefaultCacheTTL = 24 * time.Hour

func GetDefaultCacheFile(configDir string) string {
	return filepath.Join(configDir, "cache.json")
}

// NewConfiguredManager creates a manager using the persisted autodiscovery
// settings from configDir and all built-in discovery sources.
func NewConfiguredManager(configDir string) (*Manager, error) {
	config, err := LoadConfig(GetConfigFile(configDir))
	if err != nil {
		return nil, err
	}
	return NewManagerWithAllSourcesConfig(GetDefaultCacheFile(configDir), config), nil
}

func DeleteCacheFile(configDir string) error {
	cacheFile := GetDefaultCacheFile(configDir)
	err := os.Remove(cacheFile)

	if os.IsNotExist(err) {
		return nil
	}

	return err
}
