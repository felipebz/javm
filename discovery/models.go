package discovery

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type JDK struct {
	Path         string `json:"path"`
	Version      string `json:"version"`
	Vendor       string `json:"vendor"`
	Architecture string `json:"architecture"`
	Source       string `json:"source"`
	Identifier   string `json:"identifier"`
}

type Cache struct {
	LastUpdated time.Time `json:"last_updated"`
	JDKs        []JDK     `json:"jdks"`
}

func (c *Cache) SaveCache(cacheFile string) error {
	if err := os.MkdirAll(filepath.Dir(cacheFile), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cacheFile, data, 0644)
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
		return nil, err
	}

	return &cache, nil
}

func (c *Cache) IsCacheValid(ttl time.Duration) bool {
	return !c.LastUpdated.IsZero() && time.Since(c.LastUpdated) < ttl
}
