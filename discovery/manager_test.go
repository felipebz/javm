package discovery

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

type mockSource struct {
	name    string
	enabled bool
	jdks    []JDK
}

func (d *mockSource) Name() string {
	return d.name
}

func (d *mockSource) Enabled(config *Config) bool {
	return d.enabled && config.Enabled
}

func (d *mockSource) Discover() ([]JDK, error) {
	return d.jdks, nil
}

func TestNewManager(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "javm-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cacheFile := filepath.Join(tempDir, "cache.json")
	cacheTTL := 1 * time.Hour

	// Test with default parameters
	d := NewManager(cacheFile, cacheTTL)
	if d == nil {
		t.Error("Manager should not be nil")
	}
	if d.CacheFile != cacheFile {
		t.Error("Cache file should match")
	}
	if d.Config.CacheTTL != cacheTTL {
		t.Error("Cache TTL should match")
	}
	if !d.Config.Enabled {
		t.Error("Manager should be enabled by default")
	}
	if len(d.sources) != 0 {
		t.Error("Sources should be empty initially")
	}
}

func TestNewManagerWithConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "javm-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cacheFile := filepath.Join(tempDir, "cache.json")
	config := &Config{
		Enabled: false,
		Sources: map[string]bool{
			"JAVA_HOME": false,
		},
		CacheTTL: 2 * time.Hour,
	}

	d := NewManagerWithConfig(cacheFile, config)
	if d == nil {
		t.Error("Manager should not be nil")
	}
	if d.CacheFile != cacheFile {
		t.Error("Cache file should match")
	}
	if d.Config != config {
		t.Error("Config should match")
	}
	if d.Config.Enabled {
		t.Error("Manager should be disabled")
	}
	if d.Config.CacheTTL != 2*time.Hour {
		t.Error("Cache TTL should match")
	}
	if len(d.sources) != 0 {
		t.Error("Sources should be empty initially")
	}
}

func TestManager_RegisterSource(t *testing.T) {
	d := &Manager{
		sources: []Source{},
	}

	mockD := &mockSource{name: "Mock"}
	d.RegisterSource(mockD)

	if len(d.sources) != 1 {
		t.Error("Should have 1 source")
	}
	if d.sources[0] != mockD {
		t.Error("Manager should be registered")
	}
}

func TestManager_DiscoverAll(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "javm-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cacheFile := filepath.Join(tempDir, "cache.json")

	d := &Manager{
		CacheFile: cacheFile,
		Config: &Config{
			Enabled:  true,
			CacheTTL: 1 * time.Hour,
		},
		sources: []Source{},
	}

	jdk1 := JDK{
		Path:         "/path/to/jdk1",
		Version:      "17.0.2",
		Vendor:       "Oracle",
		Architecture: "x64",
		Source:       "Mock1",
	}
	jdk2 := JDK{
		Path:         "/path/to/jdk2",
		Version:      "11.0.14",
		Vendor:       "OpenJDK",
		Architecture: "x64",
		Source:       "Mock2",
	}
	jdk3 := JDK{
		Path:         "/path/to/jdk1", // Duplicate path
		Version:      "17.0.2",
		Vendor:       "Oracle",
		Architecture: "x64",
		Source:       "Mock3",
	}

	d.RegisterSource(&mockSource{
		name:    "Mock1",
		enabled: true,
		jdks:    []JDK{jdk1},
	})
	d.RegisterSource(&mockSource{
		name:    "Mock2",
		enabled: true,
		jdks:    []JDK{jdk2},
	})
	d.RegisterSource(&mockSource{
		name:    "Mock3",
		enabled: true,
		jdks:    []JDK{jdk3},
	})
	d.RegisterSource(&mockSource{
		name:    "Mock4",
		enabled: false, // Disabled
		jdks:    []JDK{},
	})

	jdks, err := d.DiscoverAll()
	if err != nil {
		t.Error("DiscoverAll should not return error")
	}
	if len(jdks) != 2 {
		t.Error("Should find 2 unique JDKs")
	}

	jdk1Found := false
	jdk2Found := false
	for _, jdk := range jdks {
		if jdk.Path == "/path/to/jdk1" {
			if jdk.Version != "17.0.2" {
				t.Error("Version should match")
			}
			if jdk.Vendor != "Oracle" {
				t.Error("Vendor should match")
			}
			if jdk.Architecture != "x64" {
				t.Error("Architecture should match")
			}
			if jdk.Source != "Mock1" {
				t.Error("Source should match")
			}
			jdk1Found = true
		} else if jdk.Path == "/path/to/jdk2" {
			if jdk.Version != "11.0.14" {
				t.Error("Version should match")
			}
			if jdk.Vendor != "OpenJDK" {
				t.Error("Vendor should match")
			}
			if jdk.Architecture != "x64" {
				t.Error("Architecture should match")
			}
			if jdk.Source != "Mock2" {
				t.Error("Source should match")
			}
			jdk2Found = true
		}
	}
	if !jdk1Found {
		t.Error("Should find JDK 1")
	}
	if !jdk2Found {
		t.Error("Should find JDK 2")
	}

	cache := &Cache{
		LastUpdated: time.Now(),
		JDKs:        jdks,
	}
	err = cache.SaveCache(cacheFile)
	if err != nil {
		t.Error("SaveCache should not return error")
	}

	jdksFromCache, err := d.DiscoverAll()
	if err != nil {
		t.Error("DiscoverAll should not return error")
	}
	if len(jdksFromCache) != 2 {
		t.Error("Should find 2 JDKs from cache")
	}

	// Test with disabled source
	d.Config.Enabled = false
	jdks, err = d.DiscoverAll()
	if err != nil {
		t.Error("DiscoverAll should not return error")
	}
	if len(jdks) != 0 {
		t.Error("Should find no JDKs when disabled")
	}
}
