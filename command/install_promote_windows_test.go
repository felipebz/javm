//go:build windows

package command

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

// openWithoutDeleteSharing opens path the way a virus scanner often does:
// while the handle is open, Windows refuses to rename its directory.
func openWithoutDeleteSharing(t *testing.T, path string) windows.Handle {
	t.Helper()
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	handle, err := windows.CreateFile(name, windows.GENERIC_READ, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatal(err)
	}
	return handle
}

func stagedDir(t *testing.T) (source, destination, file string) {
	t.Helper()
	parent := t.TempDir()
	source = filepath.Join(parent, "staging", "jdk")
	if err := os.MkdirAll(filepath.Join(source, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	file = filepath.Join(source, "bin", "java.exe")
	if err := os.WriteFile(file, []byte("java"), 0o755); err != nil {
		t.Fatal(err)
	}
	return source, filepath.Join(parent, "temurin@21.0.4"), file
}

func TestPromoteWaitsForAFileHeldOpenInTheStagedJDK(t *testing.T) {
	source, destination, file := stagedDir(t)
	handle := openWithoutDeleteSharing(t, file)
	released := make(chan struct{})
	go func() {
		time.Sleep(300 * time.Millisecond)
		_ = windows.CloseHandle(handle)
		close(released)
	}()
	defer func() { <-released }()

	if err := promoteNoReplace(source, destination); err != nil {
		t.Fatalf("promoteNoReplace() = %v, want success once the file is released", err)
	}
	if _, err := os.Stat(filepath.Join(destination, "bin", "java.exe")); err != nil {
		t.Fatalf("promoted JDK is missing java.exe: %v", err)
	}
}

func TestPromoteFailsAtOnceWhenTheDestinationExists(t *testing.T) {
	source, destination, _ := stagedDir(t)
	if err := os.Mkdir(destination, 0o755); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	err := promoteNoReplace(source, destination)
	if !errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		t.Fatalf("promoteNoReplace() = %v, want ERROR_ALREADY_EXISTS", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("promoteNoReplace() took %s for an existing destination, want no retries", elapsed)
	}
}
