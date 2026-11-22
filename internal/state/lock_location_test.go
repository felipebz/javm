package state

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestWithFileLockKeepsLockFilesInLocksDirectory(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "default-version")
	if err := WithFileLock(target, func() error { return os.WriteFile(target, []byte("21"), 0o600) }); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	if !slices.Equal(names, []string{".locks", "default-version"}) {
		t.Fatalf("entries = %q, want only .locks and default-version", names)
	}
	if _, err := os.Stat(filepath.Join(dir, ".locks", "default-version.lock")); err != nil {
		t.Fatalf("lock file is not in .locks: %v", err)
	}
}
