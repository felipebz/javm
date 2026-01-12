package discovery

import (
	"fmt"
	"time"
)

type Manager struct {
	CacheFile string
	Config    *Config
	sources   []Source
}

func NewManager(cacheFile string, cacheTTL time.Duration) *Manager {
	return &Manager{
		CacheFile: cacheFile,
		Config: &Config{
			Enabled:  true,
			CacheTTL: cacheTTL,
		},
		sources: []Source{},
	}
}

func NewManagerWithConfig(cacheFile string, config *Config) *Manager {
	return &Manager{
		CacheFile: cacheFile,
		Config:    config,
	}
}

func NewManagerWithAllSources(cacheFile string, cacheTTL time.Duration) *Manager {
	return &Manager{
		CacheFile: cacheFile,
		Config: &Config{
			Enabled:  true,
			CacheTTL: cacheTTL,
		},
		sources: []Source{
			NewSystemSource(),
			NewJabbaSource(),
			NewGradleSource(),
			NewIntelliJSource(),
			NewJavmSource(),
		},
	}
}

func (d *Manager) RegisterSource(source Source) {
	d.sources = append(d.sources, source)
}

func (d *Manager) DiscoverAll() ([]JDK, error) {
	if !d.Config.Enabled {
		return []JDK{}, nil
	}

	cache, err := LoadCache(d.CacheFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load cache: %w", err)
	}

	if cache.IsCacheValid(d.Config.CacheTTL) {
		return cache.JDKs, nil
	}

	var allJDKs []JDK

	for _, source := range d.sources {
		if d.Config.IsSourceEnabled(source.Name()) {
			jdks, err := source.Discover()
			if err != nil {
				return nil, fmt.Errorf("failed to discover from %s: %w", source.Name(), err)
			}
			allJDKs = append(allJDKs, jdks...)
		}
	}

	uniqueJDKs := DeduplicateJDKs(allJDKs)

	cache.JDKs = uniqueJDKs
	cache.LastUpdated = time.Now()
	if err := cache.SaveCache(d.CacheFile); err != nil {
		return nil, fmt.Errorf("failed to save cache: %w", err)
	}

	return uniqueJDKs, nil
}
