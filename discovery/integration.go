package discovery

import (
	"path/filepath"
	"time"
)

const DefaultCacheTTL = 24 * time.Hour

func GetDefaultCacheFile(configDir string) string {
	return filepath.Join(configDir, "cache.json")
}
