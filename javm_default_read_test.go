package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	legacyWarning   = "Warning: using legacy default.alias as the default"
	ignoredWarning  = "Warning: ignoring legacy default.alias because default-version is set"
	noDefaultSetMsg = "Error: not found: no default JDK is set; set one with 'javm default <selector>'\n"
)

func writeHomeFile(t *testing.T, home, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(home, name), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestDefaultWithoutArgumentPrintsDefault(t *testing.T) {
	_, home := isolatedWorkspace(t)
	writeHomeFile(t, home, "default-version", "21")
	for _, args := range [][]string{{"default"}, {"alias", "default"}} {
		stdout, stderr, code := executeRoot(t, args...)
		if stdout != "21\n" || stderr != "" || code != exitSuccess {
			t.Fatalf("%v = %q, %q, exit %d; want 21 and nothing on stderr", args, stdout, stderr, code)
		}
	}
}

func TestDefaultWithoutArgumentHonorsLegacyFile(t *testing.T) {
	_, home := isolatedWorkspace(t)
	writeHomeFile(t, home, "default.alias", "17")
	for _, args := range [][]string{{"default"}, {"alias", "default"}} {
		stdout, stderr, code := executeRoot(t, args...)
		if stdout != "17\n" || strings.Count(stderr, legacyWarning) != 1 || code != exitSuccess {
			t.Fatalf("%v = %q, %q, exit %d; want 17 and one migration warning", args, stdout, stderr, code)
		}
	}

	writeHomeFile(t, home, "default-version", "21")
	stdout, stderr, code := executeRoot(t, "default")
	if stdout != "21\n" || strings.Count(stderr, ignoredWarning) != 1 || code != exitSuccess {
		t.Fatalf("default = %q, %q, exit %d; want 21 and one ignoring warning", stdout, stderr, code)
	}
}

func TestDefaultWithoutArgumentWhenNothingIsSet(t *testing.T) {
	isolatedWorkspace(t)
	for _, args := range [][]string{{"default"}, {"alias", "default"}} {
		stdout, stderr, code := executeRoot(t, args...)
		if stdout != "" || stderr != noDefaultSetMsg || code != exitNotFound {
			t.Fatalf("%v = %q, %q, exit %d; want %q and exit 3", args, stdout, stderr, code, noDefaultSetMsg)
		}
	}
}

func TestDefaultWithoutArgumentRunsNoDiscovery(t *testing.T) {
	_, home := isolatedWorkspace(t)
	writeHomeFile(t, home, "default-version", "21")
	if err := os.MkdirAll(filepath.Join(home, ".locks", "cache.json.lock"), 0o700); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := executeRoot(t, "default")
	if stdout != "21\n" || stderr != "" || code != exitSuccess {
		t.Fatalf("default = %q, %q, exit %d; want 21 without a cache warning", stdout, stderr, code)
	}
}

func TestDefaultWithTwoArgumentsIsUsageError(t *testing.T) {
	isolatedWorkspace(t)
	_, stderr, code := executeRoot(t, "default", "a", "b")
	if code != exitUsage || !strings.HasSuffix(stderr, "Run 'javm default --help' for usage.\n") {
		t.Fatalf("default a b = %q, exit %d; want the usage error with the help hint", stderr, code)
	}
}
