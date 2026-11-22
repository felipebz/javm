package command

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felipebz/javm/cfg"
	"github.com/spf13/cobra"
)

const cacheSaveWarning = "could not save the discovery cache"

// unwritableCache uses the real discovery limited to javm's own JDKs, with a
// discovery cache that can be neither saved nor cleared.
func unwritableCache(t *testing.T) {
	t.Helper()
	t.Setenv("JAVM_HOME", t.TempDir())
	config := `{"enabled":true,"sources":{"system":false,"gradle":false,"intellij":false,"jabba":false},"cache_ttl":3600000000000}`
	if err := os.MkdirAll(filepath.Join(cfg.Dir(), "autodiscover"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg.Dir(), "autodiscover", "config.json"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	writeManagedJDK(t, "temurin@21.0.4", "21.0.4+7", "")
	breakCacheInvalidation(t)
}

func TestReadingCommandsWorkWithUnwritableCache(t *testing.T) {
	for name, args := range map[string][]string{"which": {"21"}, "ls": nil} {
		t.Run(name, func(t *testing.T) {
			unwritableCache(t)
			ctx, logs := contextWithLogBuffer(context.Background())
			cmd := map[string]func() *cobra.Command{"which": NewWhichCommand, "ls": NewLsCommand}[name]()
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&bytes.Buffer{})
			cmd.SetArgs(append([]string{}, args...))
			if err := cmd.ExecuteContext(ctx); err != nil {
				t.Fatalf("%s error = %v, want success", name, err)
			}
			if !strings.Contains(out.String(), "temurin@21.0.4") {
				t.Fatalf("%s output = %q, want the managed JDK", name, out.String())
			}
			if got := countLines(logs, cacheSaveWarning); got != 1 {
				t.Fatalf("logs = %q, want one cache save warning", logs.String())
			}
		})
	}
}

func TestAliasWorksWithUnwritableCache(t *testing.T) {
	unwritableCache(t)
	failSymlinks(t, func(string) bool { return false })
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
	if countLines(logs, cacheSaveWarning) > 1 || countLines(logs, cacheWarning) > 1 {
		t.Fatalf("logs = %q, want each cache warning at most once", logs.String())
	}
}

func TestDiscoverRefreshFailsWithUnwritableCache(t *testing.T) {
	unwritableCache(t)
	cmd := NewDiscoverCommand()
	cmd.SetArgs([]string{"refresh"})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SilenceUsage = true
	if err := cmd.ExecuteContext(context.Background()); err == nil || !strings.Contains(err.Error(), "failed to save cache") {
		t.Fatalf("discover refresh error = %v, want the cache save error", err)
	}
}

func TestUseDefaultWorksWithUnwritableCache(t *testing.T) {
	unwritableCache(t)
	if err := SetDefaultVersion("21"); err != nil {
		t.Fatal(err)
	}
	ctx, logs := contextWithLogBuffer(context.Background())
	if records := useDefaultRecords(t, ctx); !strings.Contains(records, "temurin@21.0.4") {
		t.Fatalf("use --default records = %q, want the default activated", records)
	}
	if got := countLines(logs, cacheSaveWarning); got != 1 {
		t.Fatalf("logs = %q, want one cache save warning", logs.String())
	}
}
