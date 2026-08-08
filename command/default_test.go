package command

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSetDefaultVersionValidatesAndPersistsSelector(t *testing.T) {
	home := t.TempDir()
	t.Setenv("JAVM_HOME", home)

	if err := SetDefaultVersion("  temurin@21  "); err != nil {
		t.Fatalf("SetDefaultVersion() returned error: %v", err)
	}
	selector, err := readDefaultVersion()
	if err != nil {
		t.Fatalf("readDefaultVersion() returned error: %v", err)
	}
	if selector != "temurin@21" {
		t.Fatalf("selector = %q, want %q", selector, "temurin@21")
	}
	if err := SetDefaultVersion("zulu@17"); err != nil {
		t.Fatalf("SetDefaultVersion() could not replace existing state: %v", err)
	}
	selector, err = readDefaultVersion()
	if err != nil {
		t.Fatalf("readDefaultVersion() after replacement returned error: %v", err)
	}
	if selector != "zulu@17" {
		t.Fatalf("selector after replacement = %q, want %q", selector, "zulu@17")
	}

	if runtime.GOOS != "windows" {
		info, err := os.Stat(filepath.Join(home, "default-version"))
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != 0o600 {
			t.Fatalf("default-version permissions = %o, want 600", got)
		}
	}
}

func TestSetDefaultVersionRejectsShellPayloadsWithoutChangingState(t *testing.T) {
	home := t.TempDir()
	t.Setenv("JAVM_HOME", home)
	if err := SetDefaultVersion("temurin@17"); err != nil {
		t.Fatal(err)
	}

	invalid := []string{
		"",
		"17\ncommand",
		"17\rcommand",
		"17\x00command",
		"17; command",
		"$(command)",
		"`command`",
	}
	for _, selector := range invalid {
		t.Run(strings.ReplaceAll(selector, "\x00", "NUL"), func(t *testing.T) {
			if err := SetDefaultVersion(selector); err == nil {
				t.Fatalf("SetDefaultVersion(%q) did not return an error", selector)
			}
			persisted, err := readDefaultVersion()
			if err != nil {
				t.Fatal(err)
			}
			if persisted != "temurin@17" {
				t.Fatalf("persisted selector = %q after rejected input", persisted)
			}
		})
	}
}

func TestReadDefaultVersionRejectsTamperedFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("JAVM_HOME", home)
	path := filepath.Join(home, "default-version")
	if err := os.WriteFile(path, []byte("17\necho compromised"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := readDefaultVersion(); err == nil || errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected invalid persisted selector error, got %v", err)
	}
}
