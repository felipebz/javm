package discovery

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDiscoveryCache_SaveAndLoadCache(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "javm-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cacheFile := filepath.Join(tempDir, "cache.json")

	now := time.Now()
	cache := &Cache{
		LastUpdated: now,
		JDKs: []JDK{
			{
				Path:           "/path/to/jdk1",
				Version:        "17.0.2",
				Vendor:         "Oracle",
				Implementation: "JDK",
				Architecture:   "x64",
				Source:         "test",
			},
			{
				Path:           "/path/to/jdk2",
				Version:        "11.0.14",
				Vendor:         "OpenJDK",
				Implementation: "JDK",
				Architecture:   "x64",
				Source:         "test",
			},
		},
	}

	err = cache.SaveCache(cacheFile)
	if err != nil {
		t.Errorf("SaveCache should not return error: %v", err)
	}
	if _, err := os.Stat(cacheFile); os.IsNotExist(err) {
		t.Error("Cache file should exist after SaveCache")
	}

	loadedCache, err := LoadCache(cacheFile)
	if err != nil {
		t.Errorf("LoadCache should not return error: %v", err)
	}
	if cache.LastUpdated.Unix() != loadedCache.LastUpdated.Unix() {
		t.Error("LastUpdated should match")
	}
	if len(loadedCache.JDKs) != 2 {
		t.Error("Should load 2 JDKs")
	}
	if cache.JDKs[0].Path != loadedCache.JDKs[0].Path {
		t.Error("Path should match")
	}
	if cache.JDKs[0].Version != loadedCache.JDKs[0].Version {
		t.Error("Version should match")
	}
	if cache.JDKs[0].Vendor != loadedCache.JDKs[0].Vendor {
		t.Error("Vendor should match")
	}
	if cache.JDKs[0].Implementation != loadedCache.JDKs[0].Implementation {
		t.Error("Implementation should match")
	}
	if cache.JDKs[0].Architecture != loadedCache.JDKs[0].Architecture {
		t.Error("Architecture should match")
	}
	if cache.JDKs[0].Source != loadedCache.JDKs[0].Source {
		t.Error("Source should match")
	}

	nonExistentFile := filepath.Join(tempDir, "non-existent.json")
	loadedCache, err = LoadCache(nonExistentFile)
	if err != nil {
		t.Errorf("LoadCache should not return error for non-existent file: %v", err)
	}
	if !loadedCache.LastUpdated.IsZero() {
		t.Error("LastUpdated should be zero for non-existent file")
	}
	if len(loadedCache.JDKs) != 0 {
		t.Error("JDKs should be empty for non-existent file")
	}

	nonExistentDir := filepath.Join(tempDir, "non-existent-dir", "cache.json")
	err = cache.SaveCache(nonExistentDir)
	if err != nil {
		t.Errorf("SaveCache should not return error for non-existent directory: %v", err)
	}
	if _, err := os.Stat(nonExistentDir); os.IsNotExist(err) {
		t.Error("Cache file should exist after SaveCache with non-existent directory")
	}
}

func TestDiscoveryCache_IsCacheValid(t *testing.T) {
	cache := &Cache{
		LastUpdated: time.Time{},
		JDKs:        []JDK{},
	}
	if cache.IsCacheValid(time.Hour) {
		t.Error("Cache should be invalid with zero LastUpdated")
	}

	cache = &Cache{
		LastUpdated: time.Now(),
		JDKs:        []JDK{},
	}
	if !cache.IsCacheValid(time.Hour) {
		t.Error("Cache should be valid with recent LastUpdated")
	}

	cache = &Cache{
		LastUpdated: time.Now().Add(-2 * time.Hour),
		JDKs:        []JDK{},
	}
	if cache.IsCacheValid(time.Hour) {
		t.Error("Cache should be invalid with old LastUpdated")
	}
}
