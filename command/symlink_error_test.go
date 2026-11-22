package command

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
)

func assertNamesOnce(t *testing.T, err error, paths ...string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error")
	}
	message := err.Error()
	for _, path := range paths {
		if got := strings.Count(message, path); got != 1 {
			t.Errorf("message %q names %q %d times, want once", message, path, got)
		}
	}
	if strings.Contains(message, `\\`) {
		t.Errorf("message %q contains doubled backslashes", message)
	}
}

func TestCreateSymlinkErrorNamesEachPathOnce(t *testing.T) {
	const link, target = `C:\Users\me\.javm\jdk\default`, `C:\Users\me\.javm\jdk\temurin@21.0.1`
	t.Cleanup(setupRemoveError(nil))
	original := createSymlink
	createSymlink = func(oldname, newname string) error {
		return &os.LinkError{Op: "symlink", Old: oldname, New: newname, Err: errPrivilegeNotHeld}
	}
	t.Cleanup(func() { createSymlink = original })
	forceGOOS(t, "windows")

	err := replaceSymlink(target, link)
	assertNamesOnce(t, err, link, target)
	if want := "create symlink " + link + " -> " + target + ": " + errPrivilegeNotHeld.Error(); !strings.Contains(err.Error(), want) {
		t.Fatalf("message %q, want %q", err, want)
	}
	var linkErr *os.LinkError
	var refreshErr *linkRefreshError
	if !errors.As(err, &linkErr) || !errors.As(err, &refreshErr) || !errors.Is(err, syscall.Errno(1314)) {
		t.Fatalf("error %v lost its *os.LinkError, *linkRefreshError or errno", err)
	}
	if symlinkHint(err) == "" {
		t.Fatal("symlinkHint() is empty for the privilege error")
	}
}

func TestRemoveSymlinkErrorNamesThePathOnce(t *testing.T) {
	const link = `C:\Users\me\.javm\jdk\default`
	cause := syscall.Errno(5)
	t.Cleanup(setupRemoveError(&fs.PathError{Op: "remove", Path: link, Err: cause}))

	err := replaceSymlink(`C:\Users\me\.javm\jdk\temurin@21.0.1`, link)
	assertNamesOnce(t, err, link)
	if want := "remove existing symlink " + link + ": " + cause.Error(); !strings.Contains(err.Error(), want) {
		t.Fatalf("message %q, want %q", err, want)
	}
	var pathErr *fs.PathError
	if !errors.As(err, &pathErr) || !errors.Is(err, cause) {
		t.Fatalf("error %v lost its *fs.PathError", err)
	}
}

func linkSystemJDK(t *testing.T, dir string) error {
	t.Helper()
	t.Setenv("JAVM_HOME", t.TempDir())
	makeFakeJDK(t, dir, runtime.GOOS == "darwin")
	failAllSymlinks(t)
	forceGOOS(t, "windows")
	return link(context.Background(), "system@21", dir)
}

func TestLinkSystemJDKErrorNamesEachPathOnce(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "external-jdk")
	err := linkSystemJDK(t, dir)
	source := filepath.Join(os.Getenv("JAVM_HOME"), "jdk", "system@21")
	assertNamesOnce(t, err, dir, source)
	if want := "create symlink " + source + " -> " + dir + ": "; !strings.Contains(err.Error(), want) {
		t.Fatalf("message %q, want %q", err, want)
	}
	if !strings.Contains(err.Error(), windowsSymlinkHint) || !errors.Is(err, errPrivilegeNotHeld) {
		t.Fatalf("message %q, want the Windows hint", err)
	}
	var linkErr *os.LinkError
	if !errors.As(err, &linkErr) {
		t.Fatalf("error %v lost its *os.LinkError", err)
	}
}

func TestSymlinkErrorSanitizesPaths(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("control characters are not allowed in Windows file names")
	}
	err := linkSystemJDK(t, filepath.Join(t.TempDir(), "jdk\x1b[31m"))
	if err == nil || strings.ContainsRune(err.Error(), '\x1b') {
		t.Fatalf("error = %q, want the control character removed", err)
	}
}
