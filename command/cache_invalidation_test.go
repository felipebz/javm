package command

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/felipebz/javm/cfg"
	"github.com/felipebz/javm/discovery"
)

const cacheWarning = "could not clear the discovery cache"

// breakCacheInvalidation makes the lock of cache.json a directory, so clearing
// the cache fails while every other file in JAVM_HOME stays writable.
func breakCacheInvalidation(t *testing.T) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(cfg.Dir(), ".locks", "cache.json.lock"), 0o700); err != nil {
		t.Fatal(err)
	}
}

func TestLinkLatestWarnsWhenCacheCannotBeCleared(t *testing.T) {
	t.Setenv("JAVM_HOME", t.TempDir())
	writeManagedJDK(t, "temurin@21.0.4", "21.0.4+7", "")
	failSymlinks(t, func(string) bool { return false })
	breakCacheInvalidation(t)
	ctx, logs := contextWithLogBuffer(context.Background())

	if err := linkLatest(ctx); err != nil {
		t.Fatalf("linkLatest() error = %v, want nil", err)
	}
	if got := countLines(logs, cacheWarning); got != 1 || !strings.Contains(logs.String(), "javm discover refresh") {
		t.Fatalf("logs = %q, want one cache warning suggesting 'javm discover refresh'", logs.String())
	}
}

func TestLinkFailureAndCacheFailureAreBothWarnings(t *testing.T) {
	t.Setenv("JAVM_HOME", t.TempDir())
	writeManagedJDK(t, "temurin@21.0.4", "21.0.4+7", "")
	failAllSymlinks(t)
	forceGOOS(t, "windows")
	breakCacheInvalidation(t)
	ctx, logs := contextWithLogBuffer(context.Background())

	if err := warnLinkFailures(ctx, linkLatest(ctx)); err != nil {
		t.Fatalf("warnLinkFailures(linkLatest()) = %v, want nil", err)
	}
	if links, caches := len(symlinkWarnings(logs)), countLines(logs, cacheWarning); links != 1 || caches != 1 {
		t.Fatalf("logs = %q, want one link warning and one cache warning", logs.String())
	}
}

func TestAliasSucceedsWhenCacheCannotBeCleared(t *testing.T) {
	t.Setenv("JAVM_HOME", t.TempDir())
	managedDiscoveryOnly(t)
	writeManagedJDK(t, "temurin@21.0.4", "21.0.4+7", "")
	failSymlinks(t, func(string) bool { return false })
	breakCacheInvalidation(t)
	ctx, logs := contextWithLogBuffer(context.Background())

	cmd := NewAliasCommand()
	cmd.SetArgs([]string{"work", "21"})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	if err := cmd.ExecuteContext(ctx); err != nil {
		t.Fatalf("alias work 21 error = %v, want success", err)
	}
	if value, err := readAlias("work"); err != nil || value != "21" {
		t.Fatalf("alias work = %q, %v", value, err)
	}
	if got := countLines(logs, cacheWarning); got != 1 {
		t.Fatalf("logs = %q, want one cache warning", logs.String())
	}
}

func TestDiscoverRefreshStillFailsWhenCacheCannotBeWritten(t *testing.T) {
	t.Setenv("JAVM_HOME", t.TempDir())
	breakCacheInvalidation(t)
	original := newManagerWithAllSources
	// No sources: the real ones would scan the user's home directory.
	newManagerWithAllSources = func(configDir string, forceRefresh bool, warn func(error)) (discoverRunner, error) {
		manager := discovery.NewManager(discovery.GetDefaultCacheFile(configDir), time.Hour)
		manager.IgnoreCache = forceRefresh
		return manager, nil
	}
	t.Cleanup(func() { newManagerWithAllSources = original })

	cmd := NewDiscoverCommand()
	cmd.SetArgs([]string{"refresh"})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SilenceUsage = true
	if err := cmd.ExecuteContext(context.Background()); err == nil {
		t.Fatal("discover refresh succeeded although the cache could not be written")
	}
}
