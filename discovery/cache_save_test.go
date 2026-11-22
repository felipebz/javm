package discovery

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDiscoverAllReturnsJDKsWhenCacheCannotBeSaved(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".locks", "cache.json.lock"), 0o700); err != nil {
		t.Fatal(err)
	}
	d := NewManager(filepath.Join(dir, "cache.json"), time.Hour)
	d.RegisterSource(&mockSource{name: "Mock", jdks: []JDK{{Path: "/jdk", Version: "21.0.1"}}})

	jdks, err := d.DiscoverAll(context.Background())
	var saveErr *CacheSaveError
	if !errors.As(err, &saveErr) {
		t.Fatalf("DiscoverAll() error = %v, want a *CacheSaveError", err)
	}
	if len(jdks) != 1 || jdks[0].Path != "/jdk" {
		t.Fatalf("DiscoverAll() = %#v, want the discovered JDK despite the save failure", jdks)
	}
}
