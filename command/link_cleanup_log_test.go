package command

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
)

func cleanupLogLine(t *testing.T, home string) (string, string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks requires privileges on Windows")
	}
	t.Setenv("JAVM_HOME", home)
	target := filepath.Join(home, "jdk", "temurin@21.0.4")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(home, "jdk", "temurin@21.0")); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(target); err != nil {
		t.Fatal(err)
	}
	var logs bytes.Buffer
	logger := log.New()
	logger.SetOutput(&logs)
	logger.SetFormatter(&log.TextFormatter{DisableQuote: true, DisableTimestamp: true})
	ctx := WithRuntime(context.Background(), Runtime{Logger: logger, Err: &bytes.Buffer{}})

	if err := linkLatest(ctx); err != nil {
		t.Fatal(err)
	}
	for line := range strings.Lines(logs.String()) {
		if strings.Contains(line, " -/> ") {
			return strings.TrimSuffix(line, "\n"), target
		}
	}
	t.Fatalf("logs = %q, want a removed link line", logs.String())
	return "", ""
}

func TestLinkLatestCleanupLogShowsOldTarget(t *testing.T) {
	line, target := cleanupLogLine(t, t.TempDir())
	if want := "temurin@21.0 -/> " + target; !strings.Contains(line, want) {
		t.Fatalf("log line = %q, want %q", line, want)
	}
}

func TestLinkLatestCleanupLogIsSanitized(t *testing.T) {
	line, _ := cleanupLogLine(t, filepath.Join(t.TempDir(), "home\x1b[31m"))
	if strings.ContainsRune(line, '\x1b') || !strings.Contains(line, "home[31m") {
		t.Fatalf("log line = %q, want the target without control characters", line)
	}
}
