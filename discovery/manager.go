package discovery

import (
	"errors"
	"fmt"
	"time"
)

type Manager struct {
	CacheFile   string
	Config      *Config
	IgnoreCache bool
	Warn        func(error)
	sources     []Source
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
	config := DefaultConfig()
	config.CacheTTL = cacheTTL
	return NewManagerWithAllSourcesConfig(cacheFile, config)
}

func NewManagerWithAllSourcesConfig(cacheFile string, config *Config) *Manager {
	manager := NewManagerWithConfig(cacheFile, config)
	manager.sources = allSources()
	return manager
}

func allSources() []Source {
	return []Source{
		NewSystemSource(),
		NewJabbaSource(),
		NewGradleSource(),
		NewIntelliJSource(),
		NewJavmSource(),
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
		if !errors.Is(err, ErrInvalidCache) {
			return nil, fmt.Errorf("failed to load cache: %w", err)
		}
		if d.Warn != nil {
			d.Warn(fmt.Errorf("ignoring corrupt discovery cache %q: %w", d.CacheFile, err))
		}
		cache = &Cache{JDKs: []JDK{}}
	}

	configFingerprint, err := d.Config.fingerprint()
	if err != nil {
		return nil, err
	}
	if !d.IgnoreCache && cache.IsCacheValid(d.Config.CacheTTL) && cache.ConfigFingerprint == configFingerprint {
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
	cache.ConfigFingerprint = configFingerprint
	if err := cache.SaveCache(d.CacheFile); err != nil {
		return nil, fmt.Errorf("failed to save cache: %w", err)
	}

	return uniqueJDKs, nil
}
