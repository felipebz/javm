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

func DeleteCacheFile(configDir string) error {
	cacheFile := GetDefaultCacheFile(configDir)
	return os.Remove(cacheFile)
}
