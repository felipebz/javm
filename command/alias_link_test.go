package command

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/felipebz/javm/cfg"
	"github.com/felipebz/javm/discovery"
)

func runAlias(t *testing.T, args ...string) error {
	t.Helper()
	cmd := NewAliasCommand()
	cmd.SetArgs(args)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SilenceUsage = true
	return cmd.ExecuteContext(context.Background())
}

// stubDiscovery makes the full JDK listing return jdks.
func stubDiscovery(t *testing.T, jdks ...discovery.JDK) {
	t.Helper()
	cleanup := setupMockLs()
	t.Cleanup(cleanup)
	mockLsResult = jdks
	mockLsError = nil
}

func discoveredJDK(t *testing.T) discovery.JDK {
	t.Helper()
	path := filepath.Join(t.TempDir(), "java-21-openjdk")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	return discovery.JDK{Identifier: "ubuntu-system@21", Version: "21.0.4", Source: "system", Path: path}
}

func aliasLinkTarget(t *testing.T, name string) string {
	t.Helper()
	target, err := os.Readlink(filepath.Join(cfg.Dir(), "jdk", name))
	if err != nil {
		t.Fatalf("jdk/%s is not a symlink: %v", name, err)
	}
	return target
}

func TestAliasLinksDiscoveredJDK(t *testing.T) {
	t.Setenv("JAVM_HOME", t.TempDir())
	jdk := discoveredJDK(t)
	stubDiscovery(t, jdk)

	if err := runAlias(t, "work", "21"); err != nil {
		t.Fatalf("alias work 21 error = %v", err)
	}
	if value, err := readAlias("work"); err != nil || value != "21" {
		t.Fatalf("alias file = %q, %v, want 21", value, err)
	}
	if target := aliasLinkTarget(t, "work"); target != jdk.Path {
		t.Fatalf("jdk/work -> %q, want the discovered JDK %q", target, jdk.Path)
	}
}

func TestAliasLinksManagedJDK(t *testing.T) {
	t.Setenv("JAVM_HOME", t.TempDir())
	writeManagedJDK(t, "temurin@21.0.4", "21.0.4+7", "")
	managed := filepath.Join(cfg.Dir(), "jdk", "temurin@21.0.4")
	stubDiscovery(t, discovery.JDK{Identifier: "temurin@21.0.4", Version: "21.0.4+7", Source: "javm", Path: managed})

	if err := runAlias(t, "work", "temurin@21"); err != nil {
		t.Fatalf("alias work temurin@21 error = %v", err)
	}
	if target := aliasLinkTarget(t, "work"); target != managed {
		t.Fatalf("jdk/work -> %q, want %q", target, managed)
	}
}

func TestAliasRejectsSelectorWithoutJDK(t *testing.T) {
	t.Setenv("JAVM_HOME", t.TempDir())
	jdk := discoveredJDK(t)
	stubDiscovery(t, jdk)

	if err := runAlias(t, "x", "99"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("alias x 99 error = %v, want ErrNotFound", err)
	}
	if _, err := readAlias("x"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("alias file was written: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(cfg.Dir(), "jdk", "x")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("jdk/x was created: %v", err)
	}

	if err := runAlias(t, "x", "21"); err != nil {
		t.Fatalf("alias x 21 error = %v", err)
	}
	if err := runAlias(t, "x", "99"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("alias x 99 error = %v, want ErrNotFound", err)
	}
	if value, err := readAlias("x"); err != nil || value != "21" {
		t.Fatalf("alias file = %q, %v, want the old value 21", value, err)
	}
	if target := aliasLinkTarget(t, "x"); target != jdk.Path {
		t.Fatalf("jdk/x -> %q, want it unchanged at %q", target, jdk.Path)
	}
}

func TestLinkAliasRemovesOnlyStaleSymlinks(t *testing.T) {
	t.Setenv("JAVM_HOME", t.TempDir())
	jdkDir := filepath.Join(cfg.Dir(), "jdk")
	if err := os.MkdirAll(jdkDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"stale", "realdir"} {
		if err := setAlias(name, "99"); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(filepath.Join(jdkDir, "99"), filepath.Join(jdkDir, "stale")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(jdkDir, "realdir"), 0o755); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"stale", "realdir"} {
		if err := linkAlias(context.Background(), name, nil); err != nil {
			t.Fatalf("linkAlias(%s) error = %v", name, err)
		}
		if value, err := readAlias(name); err != nil || value != "99" {
			t.Fatalf("alias %s = %q, %v, want it kept", name, value, err)
		}
	}
	if _, err := os.Lstat(filepath.Join(jdkDir, "stale")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("stale jdk/stale link was kept: %v", err)
	}
	if info, err := os.Lstat(filepath.Join(jdkDir, "realdir")); err != nil || !info.IsDir() {
		t.Fatalf("real directory jdk/realdir was removed: %v", err)
	}
}

func TestLinkLatestKeepsDefaultLinkToDiscoveredJDK(t *testing.T) {
	t.Setenv("JAVM_HOME", t.TempDir())
	writeManagedJDK(t, "temurin@17.0.12", "17.0.12+7", "")
	jdk := discoveredJDK(t)
	stubDiscovery(t, jdk, discovery.JDK{Identifier: "temurin@17.0.12", Version: "17.0.12+7", Source: "javm", Path: filepath.Join(cfg.Dir(), "jdk", "temurin@17.0.12")})
	if err := runAlias(t, "default", "21"); err != nil {
		t.Fatalf("alias default 21 error = %v", err)
	}

	if err := linkLatest(context.Background()); err != nil {
		t.Fatalf("linkLatest() error = %v", err)
	}
	if target := aliasLinkTarget(t, "default"); target != jdk.Path {
		t.Fatalf("jdk/default -> %q, want the discovered JDK %q", target, jdk.Path)
	}
}
