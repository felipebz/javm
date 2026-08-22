package command

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/felipebz/javm/cfg"
	"github.com/felipebz/javm/discovery"
)

func TestLinkLatestTreatsMissingManagedDirectoryAsEmpty(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks requires privileges on Windows")
	}

	t.Setenv("JAVM_HOME", t.TempDir())
	cleanup := setupMockLs()
	defer cleanup()
	mockLsResult = nil

	if err := linkLatest(context.Background()); err != nil {
		t.Fatalf("linkLatest() error = %v", err)
	}
}

func TestLinkLatestPropagatesReadDirectoryError(t *testing.T) {
	t.Setenv("JAVM_HOME", t.TempDir())

	expected := fs.ErrPermission
	originalReadDir := readDir
	readDir = func(string) ([]os.DirEntry, error) {
		return nil, expected
	}
	t.Cleanup(func() { readDir = originalReadDir })

	err := linkLatest(context.Background())
	if !errors.Is(err, expected) {
		t.Fatalf("linkLatest() error = %v, want %v", err, expected)
	}
}

func TestLinkAndUnlinkInvalidateDiscoveryCache(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks requires privileges on Windows")
	}

	home := t.TempDir()
	t.Setenv("JAVM_HOME", home)
	target := t.TempDir()
	javaPath := filepath.FromSlash(discovery.ExpectedJavaPath(target, runtime.GOOS))
	if err := os.MkdirAll(filepath.Dir(javaPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(javaPath, []byte("java"), 0o755); err != nil {
		t.Fatal(err)
	}

	cacheFile := discovery.GetDefaultCacheFile(cfg.Dir())
	if err := os.WriteFile(cacheFile, []byte(`{"cached":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	selector := "system@21.0.1"
	if err := link(context.Background(), selector, target); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(cacheFile); !os.IsNotExist(err) {
		t.Fatalf("cache still exists after link: %v", err)
	}

	if err := os.WriteFile(cacheFile, []byte(`{"cached":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cleanup := setupMockLs()
	defer cleanup()
	mockLsResult = []discovery.JDK{{
		Identifier: selector,
		Version:    "21.0.1",
		Source:     "system",
		Path:       target,
	}}
	if err := link(context.Background(), selector, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(cacheFile); !os.IsNotExist(err) {
		t.Fatalf("cache still exists after unlink: %v", err)
	}
}
