package discovery

import (
	"path/filepath"
	"strings"
	"time"
)

const DefaultCacheTTL = 24 * time.Hour

func GetDefaultCacheFile(configDir string) string {
	return filepath.Join(configDir, "cache.json")
}

func CleanupVersionString(version string) string {
	version = strings.ReplaceAll(version, "_", ".")

	// Remove any "1.8.0" style prefix and convert to "8.0.0"
	if strings.HasPrefix(version, "1.") {
		parts := strings.Split(version, ".")
		if len(parts) >= 3 {
			version = parts[1] + "." + parts[2]
			if len(parts) == 4 {
				version += "." + parts[3]
			}
		}
	}

	version = strings.Trim(version, "\"")

	if idx := strings.IndexAny(version, " -"); idx != -1 {
		version = version[:idx]
	}

	parts := strings.Split(version, ".")
	if len(parts) == 1 {
		version += ".0.0"
	} else if len(parts) == 2 {
		version += ".0"
	}

	return version
}
