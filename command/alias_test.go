package command

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestValidateAliasName(t *testing.T) {
	for _, name := range []string{"default", "java-17", "team_jdk", "jdk.21", "A1"} {
		if err := validateAliasName(name); err != nil {
			t.Errorf("validateAliasName(%q) returned error: %v", name, err)
		}
	}

	for _, name := range []string{"", ".", "..", "../outside", `..\outside`, "path/name", `path\name`, "with space", "jdk@17", "ação"} {
		if err := validateAliasName(name); err == nil {
			t.Errorf("validateAliasName(%q) did not return an error", name)
		}
	}
}

func TestSetAliasRejectsPathTraversal(t *testing.T) {
	parent := t.TempDir()
	home := filepath.Join(parent, "javm")
	t.Setenv("JAVM_HOME", home)

	if err := setAlias("../outside", "17"); err == nil {
		t.Fatal("setAlias() did not reject path traversal")
	}
	if _, err := os.Stat(filepath.Join(parent, "outside.alias")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("outside alias file was accessed: %v", err)
	}
}

func TestSetAliasCreatesConfigurationDirectory(t *testing.T) {
	home := filepath.Join(t.TempDir(), "new", "javm")
	t.Setenv("JAVM_HOME", home)

	if err := setAlias("default", "17"); err != nil {
		t.Fatalf("setAlias() returned error: %v", err)
	}
	value, err := readAlias("default")
	if err != nil {
		t.Fatalf("readAlias() returned error: %v", err)
	}
	if value != "17" {
		t.Fatalf("readAlias() = %q, want %q", value, "17")
	}

	if runtime.GOOS != "windows" {
		info, err := os.Stat(filepath.Join(home, "default.alias"))
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != 0600 {
			t.Errorf("alias permissions = %o, want 600", got)
		}
	}
}

func TestReadAliasRejectsInvalidName(t *testing.T) {
	t.Setenv("JAVM_HOME", t.TempDir())

	if _, err := readAlias("../outside"); err == nil {
		t.Fatal("readAlias() did not reject path traversal")
	}
}
