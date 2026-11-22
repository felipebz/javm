package state

import (
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
)

func TestLocksDirectoryIsHiddenOnWindows(t *testing.T) {
	dir := t.TempDir()
	if err := WithFileLock(filepath.Join(dir, "default-version"), func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	path, err := windows.UTF16PtrFromString(filepath.Join(dir, ".locks"))
	if err != nil {
		t.Fatal(err)
	}
	attributes, err := windows.GetFileAttributes(path)
	if err != nil {
		t.Fatal(err)
	}
	if attributes&windows.FILE_ATTRIBUTE_HIDDEN == 0 {
		t.Fatalf(".locks attributes = %#x, want FILE_ATTRIBUTE_HIDDEN", attributes)
	}
}
