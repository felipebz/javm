package command

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/felipebz/javm/cfg"
	"github.com/felipebz/javm/discoapi"
	"github.com/felipebz/javm/discovery"
	log "github.com/sirupsen/logrus"
)

// writeBrokenManagedJDK creates jdk/<name> with a java whose metadata cannot be
// read: the release file lacks vendor and architecture, and java prints
// nothing (on Windows it is not a runnable executable).
func writeBrokenManagedJDK(t *testing.T, name string) string {
	t.Helper()
	dir := filepath.Join(cfg.Dir(), "jdk", name)
	home := filepath.FromSlash(discovery.ExpectedJDKDir(dir, runtime.GOOS))
	java := filepath.Join(home, "bin", "java")
	body := "#!/bin/sh\nexit 0\n"
	if runtime.GOOS == "windows" {
		java += ".exe"
		body = "not an executable"
	}
	if err := os.MkdirAll(filepath.Dir(java), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(java, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "release"), []byte("JAVA_VERSION=\"21.0.4\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// contextWithLogBuffer returns a command context whose warnings go to the
// returned buffer.
func contextWithLogBuffer(ctx context.Context) (context.Context, *bytes.Buffer) {
	var logs bytes.Buffer
	logger := log.New()
	logger.SetOutput(&logs)
	logger.SetLevel(log.WarnLevel)
	return WithRuntime(ctx, Runtime{Logger: logger, Err: &bytes.Buffer{}}), &logs
}

// brokenJDKWarnings counts the logged warnings about the broken JDK. The cause
// can repeat the JDK name (on Windows it contains java.exe's path), so count
// warning entries rather than name occurrences.
func brokenJDKWarnings(logs *bytes.Buffer) int {
	count := 0
	for line := range strings.Lines(logs.String()) {
		if strings.Contains(line, "level=warning") && strings.Contains(line, `jdk/temurin@21.0.4`) {
			count++
		}
	}
	return count
}

func TestLsManagedSkipsBrokenJDKAndWarnsOnce(t *testing.T) {
	t.Setenv("JAVM_HOME", t.TempDir())
	writeManagedJDK(t, "temurin@25.0.4", "25.0.4+7", "ok")
	writeBrokenManagedJDK(t, "temurin@21.0.4")
	ctx, logs := contextWithLogBuffer(context.Background())

	for range 2 {
		jdks, err := LsContext(ctx, true)
		if err != nil {
			t.Fatalf("LsContext() error = %v", err)
		}
		if len(jdks) != 1 || jdks[0].Identifier != "temurin@25.0.4" {
			t.Fatalf("LsContext() = %+v, want only temurin@25.0.4", jdks)
		}
	}
	if got := brokenJDKWarnings(logs); got != 1 {
		t.Fatalf("warning logged %d times, want once:\n%s", got, logs.String())
	}
}

func TestLsManagedReturnsCancellation(t *testing.T) {
	t.Setenv("JAVM_HOME", t.TempDir())
	writeBrokenManagedJDK(t, "temurin@21.0.4")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := LsContext(ctx, true); !errors.Is(err, context.Canceled) {
		t.Fatalf("LsContext() error = %v, want context.Canceled", err)
	}
}

func TestInstallSucceedsNextToBrokenJDK(t *testing.T) {
	t.Setenv("JAVM_HOME", t.TempDir())
	writeBrokenManagedJDK(t, "temurin@21.0.4")
	client := &updateClient{
		packages: []discoapi.Package{{Id: "17", JavaVersion: "17.0.12+7", Distribution: "temurin", DistributionVersion: "17.0.12"}},
		archives: map[string]string{"17": updateArchive(t, "17.0.12+7", "new")},
	}
	ctx, logs := contextWithLogBuffer(context.Background())

	cmd := NewInstallCommand(client)
	cmd.SetArgs([]string{"temurin@17"})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	if err := cmd.ExecuteContext(ctx); err != nil {
		t.Fatalf("install error = %v", err)
	}
	assertExists(t, "temurin@17.0.12", true)
	if got := brokenJDKWarnings(logs); got != 1 {
		t.Fatalf("warning logged %d times, want once:\n%s", got, logs.String())
	}
}

func TestInstallNamesBrokenDestination(t *testing.T) {
	t.Setenv("JAVM_HOME", t.TempDir())
	broken := writeBrokenManagedJDK(t, "temurin@21.0.4")
	client := &updateClient{
		packages: []discoapi.Package{{Id: "21", JavaVersion: "21.0.4+7", Distribution: "temurin", DistributionVersion: "21.0.4"}},
		archives: map[string]string{"21": updateArchive(t, "21.0.4+7", "new")},
	}

	_, err := runInstall(context.Background(), client, "temurin@21.0.4", "")
	if err == nil {
		t.Fatal("runInstall replaced or accepted a broken JDK directory")
	}
	for _, want := range []string{broken, "javm uninstall temurin@21.0.4"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not mention %q", err, want)
		}
	}
	if _, err := os.Stat(filepath.FromSlash(discovery.ExpectedJavaPath(broken, runtime.GOOS))); err != nil {
		t.Fatalf("broken directory was changed: %v", err)
	}
}

func TestUpdateIgnoresBrokenSibling(t *testing.T) {
	client := temurinUpdateFixture(t)
	writeBrokenManagedJDK(t, "temurin@21.0.4")
	ctx, logs := contextWithLogBuffer(context.Background())

	if err := runUpdate(ctx, client, &bytes.Buffer{}, "", updateOptions{}); err != nil {
		t.Fatalf("runUpdate() error = %v", err)
	}
	assertExists(t, "temurin@25.0.5", true)
	assertExists(t, "temurin@21.0.4", true)
	if got := brokenJDKWarnings(logs); got != 1 {
		t.Fatalf("warning logged %d times, want once:\n%s", got, logs.String())
	}
}

func runUninstall(t *testing.T, selector string) error {
	t.Helper()
	cmd := NewUninstallCommand()
	cmd.SetArgs([]string{selector})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SilenceUsage = true
	return cmd.ExecuteContext(context.Background())
}

func TestUninstallRemovesBrokenJDKByExactName(t *testing.T) {
	t.Setenv("JAVM_HOME", t.TempDir())
	writeBrokenManagedJDK(t, "temurin@21.0.4")

	// A selector only resolves validated JDKs; the exact directory name is
	// the way to remove a broken one.
	if err := runUninstall(t, "21"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("uninstall 21 error = %v, want ErrNotFound", err)
	}
	assertExists(t, "temurin@21.0.4", true)

	if err := runUninstall(t, "temurin@21.0.4"); err != nil {
		t.Fatalf("uninstall temurin@21.0.4 error = %v", err)
	}
	assertExists(t, "temurin@21.0.4", false)
}

func TestUninstallStillRefusesLinksAndSystemNames(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks requires privileges on Windows")
	}
	t.Setenv("JAVM_HOME", t.TempDir())
	target := writeBrokenManagedJDK(t, "temurin@21.0.4")
	link := filepath.Join(cfg.Dir(), "jdk", "temurin@22.0.1")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	systemLink := filepath.Join(cfg.Dir(), "jdk", "system@21")
	if err := os.Symlink(target, systemLink); err != nil {
		t.Fatal(err)
	}

	if err := runUninstall(t, "temurin@22.0.1"); err == nil {
		t.Fatal("uninstall removed a symlinked JDK")
	}
	if err := runUninstall(t, "system@21"); !errors.Is(err, ErrUsage) {
		t.Fatalf("uninstall system@21 error = %v, want ErrUsage", err)
	}
	for _, path := range []string{link, systemLink, target} {
		if _, err := os.Lstat(path); err != nil {
			t.Fatalf("%s was removed: %v", path, err)
		}
	}
}
