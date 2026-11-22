package command

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/felipebz/javm/cfg"
	log "github.com/sirupsen/logrus"
)

func TestRemovedAliasLinkLogShowsOldTarget(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks requires privileges on Windows")
	}
	t.Setenv("JAVM_HOME", t.TempDir())
	target := filepath.Join(cfg.Dir(), "jdk", "temurin@21.0.4")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(cfg.Dir(), "jdk", "work")); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(target); err != nil {
		t.Fatal(err)
	}
	if err := setAlias("work", "temurin@21"); err != nil {
		t.Fatal(err)
	}
	var logs bytes.Buffer
	logger := log.New()
	logger.SetOutput(&logs)
	logger.SetFormatter(&log.TextFormatter{DisableQuote: true, DisableTimestamp: true})
	ctx := WithRuntime(context.Background(), Runtime{Logger: logger, Err: &bytes.Buffer{}})

	if err := linkAlias(ctx, "work", nil); err != nil {
		t.Fatal(err)
	}
	if want := "work -/> " + target; !strings.Contains(logs.String(), want) {
		t.Fatalf("logs = %q, want %q", logs.String(), want)
	}
}
