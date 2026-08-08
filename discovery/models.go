package discovery

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var ErrInvalidCache = errors.New("invalid discovery cache")

type JDK struct {
	Path         string `json:"path"`
	Version      string `json:"version"`
	Vendor       string `json:"vendor"`
	Architecture string `json:"architecture"`
	Source       string `json:"source"`
	Identifier   string `json:"identifier"`
}

type Cache struct {
	LastUpdated       time.Time `json:"last_updated"`
	ConfigFingerprint string    `json:"config_fingerprint,omitempty"`
	JDKs              []JDK     `json:"jdks"`
}

func (c *Cache) SaveCache(cacheFile string) error {
	dir := filepath.Dir(cacheFile)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create cache directory: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("encode discovery cache: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".cache-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary discovery cache: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("restrict discovery cache permissions: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write discovery cache: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync discovery cache: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close discovery cache: %w", err)
	}
	if err := replaceCacheFile(tmpPath, cacheFile); err != nil {
		return fmt.Errorf("replace discovery cache: %w", err)
	}
	return nil
}

func LoadCache(cacheFile string) (*Cache, error) {
	if _, err := os.Stat(cacheFile); os.IsNotExist(err) {
		return &Cache{
			LastUpdated: time.Time{},
			JDKs:        []JDK{},
		}, nil
	}

	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil, err
	}

	var cache Cache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidCache, err)
	}

	return &cache, nil
}

func (c *Cache) IsCacheValid(ttl time.Duration) bool {
	return !c.LastUpdated.IsZero() && time.Since(c.LastUpdated) < ttl
}
