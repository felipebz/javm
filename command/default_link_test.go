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
	"github.com/felipebz/javm/discovery"
)

func managedDiscoveryOnly(t *testing.T) {
	t.Helper()
	original := lsFunc
	lsFunc = func(ctx context.Context) ([]discovery.JDK, error) {
		return discovery.NewJavmSource().DiscoverManaged(ctx)
	}
	t.Cleanup(func() { lsFunc = original })
}

func runDefault(t *testing.T, ctx context.Context, selector string) error {
	t.Helper()
	cmd := NewDefaultCommand()
	cmd.SetArgs([]string{selector})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SilenceUsage = true
	return cmd.ExecuteContext(ctx)
}

func writeLegacyDefaultAlias(t *testing.T, selector string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(cfg.Dir(), "default.alias"), []byte(selector), 0o600); err != nil {
		t.Fatal(err)
	}
}

func countLines(logs *bytes.Buffer, substring string) int {
	count := 0
	for line := range strings.Lines(logs.String()) {
		if strings.Contains(line, substring) {
			count++
		}
	}
	return count
}

func useDefaultRecords(t *testing.T, ctx context.Context) string {
	t.Helper()
	environment := filepath.Join(t.TempDir(), "environment")
	cmd := NewUseCommand()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--default", "--fd3", environment})
	if err := cmd.ExecuteContext(ctx); err != nil {
		t.Fatalf("use --default error = %v", err)
	}
	data, _ := os.ReadFile(environment)
	return string(data)
}

func TestDefaultCommandDrivesDefaultLink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks requires privileges on Windows")
	}
	t.Setenv("JAVM_HOME", t.TempDir())
	managedDiscoveryOnly(t)
	writeManagedJDK(t, "temurin@21.0.4", "21.0.4+7", "")
	writeManagedJDK(t, "temurin@17.0.12", "17.0.12+7", "")

	for selector, want := range map[string]string{"temurin@21": "temurin@21.0.4", "17": "temurin@17.0.12"} {
		if err := runDefault(t, context.Background(), selector); err != nil {
			t.Fatalf("default %s error = %v", selector, err)
		}
		if got := filepath.Base(aliasLinkTarget(t, "default")); got != want {
			t.Fatalf("after default %s, jdk/default -> %s, want %s", selector, got, want)
		}
	}
}

func TestLinkLatestRemovesDefaultLinkWithoutDefault(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks requires privileges on Windows")
	}
	t.Setenv("JAVM_HOME", t.TempDir())
	managedDiscoveryOnly(t)
	link := filepath.Join(cfg.Dir(), "jdk", "default")
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(cfg.Dir(), "jdk", "temurin@21.0.4"), link); err != nil {
		t.Fatal(err)
	}
	if err := linkLatest(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(link); !os.IsNotExist(err) {
		t.Fatalf("stale jdk/default was kept: %v", err)
	}
}

func TestAliasDefaultIsReserved(t *testing.T) {
	t.Setenv("JAVM_HOME", t.TempDir())
	managedDiscoveryOnly(t)
	writeManagedJDK(t, "temurin@21.0.4", "21.0.4+7", "")

	err := runAlias(t, "default", "21")
	if !errors.Is(err, ErrUsage) || !strings.Contains(err.Error(), "javm default") {
		t.Fatalf("alias default 21 error = %v, want a usage error pointing at 'javm default'", err)
	}
	if _, err := os.Stat(filepath.Join(cfg.Dir(), "default.alias")); !os.IsNotExist(err) {
		t.Fatalf("default.alias was written: %v", err)
	}

	writeLegacyDefaultAlias(t, "21")
	cmd := NewAliasCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"default"})
	if err := cmd.Execute(); err != nil || out.String() != "21\n" {
		t.Fatalf("alias default = %q, %v, want the legacy value", out.String(), err)
	}
	unalias := NewUnaliasCommand()
	unalias.SetOut(&bytes.Buffer{})
	unalias.SetArgs([]string{"default"})
	if err := unalias.Execute(); err != nil {
		t.Fatalf("unalias default error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(cfg.Dir(), "default.alias")); !os.IsNotExist(err) {
		t.Fatalf("default.alias still exists after unalias: %v", err)
	}
}

func initScript(t *testing.T, shell string) string {
	t.Helper()
	cmd := NewInitCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{shell})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

// Shell startup reads only default-version, as javm 0.17 did; the legacy file
// still drives jdk/default.
func TestLegacyDefaultAliasIsIgnoredAtShellStartup(t *testing.T) {
	t.Setenv("JAVM_HOME", t.TempDir())
	managedDiscoveryOnly(t)
	writeManagedJDK(t, "temurin@21.0.4", "21.0.4+7", "")
	writeLegacyDefaultAlias(t, "21")
	ctx, logs := contextWithLogBuffer(context.Background())

	if records := useDefaultRecords(t, ctx); records != "" || logs.Len() != 0 {
		t.Fatalf("use --default records = %q, logs = %q, want nothing", records, logs.String())
	}
	if script := initScript(t, "bash"); strings.Contains(script, "javm use --default") {
		t.Fatal("init bash runs 'javm use --default' with only the legacy default.alias")
	}
	if runtime.GOOS != "windows" {
		if err := linkLatest(ctx); err != nil {
			t.Fatal(err)
		}
		if got := filepath.Base(aliasLinkTarget(t, "default")); got != "temurin@21.0.4" {
			t.Fatalf("jdk/default -> %s, want temurin@21.0.4", got)
		}
		if got := countLines(logs, "legacy default.alias"); got != 1 {
			t.Fatalf("logs = %q, want one legacy warning from linkLatest", logs.String())
		}
	}
}

func TestUpdateKeepsBuildHeldByLegacyDefaultAlias(t *testing.T) {
	client := temurinUpdateFixture(t)
	managedDiscoveryOnly(t)
	writeLegacyDefaultAlias(t, "temurin@25.0.4")

	if err := runUpdate(context.Background(), client, &bytes.Buffer{}, "temurin@25", updateOptions{}); err != nil {
		t.Fatalf("runUpdate() error = %v", err)
	}
	assertExists(t, "temurin@25.0.5", true)
	assertExists(t, "temurin@25.0.4", true)
}

func TestUpdateNamesInvalidLegacyDefaultAlias(t *testing.T) {
	client := temurinUpdateFixture(t)
	managedDiscoveryOnly(t)
	writeLegacyDefaultAlias(t, "temurin@25.0.4\necho compromised")
	ctx, logs := contextWithLogBuffer(context.Background())

	if err := runUpdate(ctx, client, &bytes.Buffer{}, "temurin@25", updateOptions{}); err != nil {
		t.Fatalf("runUpdate() error = %v", err)
	}
	if !strings.Contains(logs.String(), "default.alias") || strings.Contains(logs.String(), "default-version:") {
		t.Fatalf("logs = %q, want the warning to name default.alias", logs.String())
	}
}

func TestDefaultVersionWinsOverLegacyDefaultAlias(t *testing.T) {
	t.Setenv("JAVM_HOME", t.TempDir())
	managedDiscoveryOnly(t)
	writeManagedJDK(t, "temurin@21.0.4", "21.0.4+7", "")
	writeManagedJDK(t, "temurin@17.0.12", "17.0.12+7", "")
	if err := SetDefaultVersion("17"); err != nil {
		t.Fatal(err)
	}
	writeLegacyDefaultAlias(t, "21")
	ctx, logs := contextWithLogBuffer(context.Background())

	if records := useDefaultRecords(t, ctx); !strings.Contains(records, "temurin@17.0.12") || logs.Len() != 0 {
		t.Fatalf("use --default records = %q, logs = %q, want default-version and no warning", records, logs.String())
	}
	_ = linkLatest(ctx)
	if got := countLines(logs, "javm unalias default"); got != 1 {
		t.Fatalf("logs = %q, want one warning naming 'javm unalias default'", logs.String())
	}
}

func TestDefaultCommandRemovesLegacyDefaultAlias(t *testing.T) {
	t.Setenv("JAVM_HOME", t.TempDir())
	managedDiscoveryOnly(t)
	writeManagedJDK(t, "temurin@21.0.4", "21.0.4+7", "")
	writeLegacyDefaultAlias(t, "17")
	failAllSymlinks(t)

	if err := runDefault(t, context.Background(), "21"); err != nil {
		t.Fatalf("default 21 error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(cfg.Dir(), "default.alias")); !os.IsNotExist(err) {
		t.Fatalf("default.alias still exists: %v", err)
	}
}

func TestDefaultCommandWarnsWhenDefaultLinkFails(t *testing.T) {
	t.Setenv("JAVM_HOME", t.TempDir())
	managedDiscoveryOnly(t)
	writeManagedJDK(t, "temurin@21.0.4", "21.0.4+7", "")
	failAllSymlinks(t)
	forceGOOS(t, "windows")
	ctx, logs := contextWithLogBuffer(context.Background())

	if err := runDefault(t, ctx, "21"); err != nil {
		t.Fatalf("default 21 error = %v, want success with a warning", err)
	}
	if data, err := os.ReadFile(defaultVersionPath()); err != nil || string(data) != "21" {
		t.Fatalf("default-version = %q, %v", data, err)
	}
	warnings := symlinkWarnings(logs)
	if len(warnings) != 1 || !strings.Contains(warnings[0], "default -> ") || !strings.Contains(warnings[0], windowsSymlinkHint) {
		t.Fatalf("link warnings = %q, want one for jdk/default with the hint", warnings)
	}
}

func TestUninstallTreatsLegacyDefaultAliasAsDefault(t *testing.T) {
	warnings := uninstallWithPins(t, map[string]string{"temurin@21.0.4": "21.0.4+7"},
		map[string]string{"default.alias": "21"}, "21")
	if len(warnings) != 1 || !strings.Contains(warnings[0], "default version 21") || strings.Contains(warnings[0], "alias default") {
		t.Fatalf("warnings = %q, want one 'default version' warning", warnings)
	}
}
