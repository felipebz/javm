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
	"github.com/felipebz/javm/discovery"
)

func setupUninstallPins(t *testing.T, jdks map[string]string, pins map[string]string) {
	t.Helper()
	t.Setenv("JAVM_HOME", t.TempDir())
	original := lsFunc
	lsFunc = func(ctx context.Context) ([]discovery.JDK, error) {
		return discovery.NewJavmSource().DiscoverManaged(ctx)
	}
	t.Cleanup(func() { lsFunc = original })
	for identifier, version := range jdks {
		writeManagedJDK(t, identifier, version, "")
	}
	for file, content := range pins {
		if err := os.WriteFile(filepath.Join(cfg.Dir(), file), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func uninstallLogged(t *testing.T, selector string) *bytes.Buffer {
	t.Helper()
	ctx, logs := contextWithLogBuffer(context.Background())
	cmd := NewUninstallCommand()
	cmd.SetArgs([]string{selector})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	if err := cmd.ExecuteContext(ctx); err != nil {
		t.Fatalf("uninstall %s error = %v", selector, err)
	}
	return logs
}

func pinWarnings(logs *bytes.Buffer) []string {
	var warnings []string
	for line := range strings.Lines(logs.String()) {
		if strings.Contains(line, "no longer matches an installed JDK") {
			warnings = append(warnings, line)
		}
	}
	return warnings
}

func uninstallWithPins(t *testing.T, jdks map[string]string, pins map[string]string, selector string) []string {
	t.Helper()
	setupUninstallPins(t, jdks, pins)
	return pinWarnings(uninstallLogged(t, selector))
}

func TestUninstallRemovesLinkOfAliasThatStopsResolving(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks requires privileges on Windows")
	}
	setupUninstallPins(t, map[string]string{"temurin@21.0.4": "21.0.4+7"}, nil)
	if err := runAlias(t, "work", "temurin@21"); err != nil {
		t.Fatal(err)
	}

	logs := uninstallLogged(t, "temurin@21")
	if _, err := os.Lstat(filepath.Join(cfg.Dir(), "jdk", "work")); !os.IsNotExist(err) {
		t.Fatalf("jdk/work still exists after uninstall: %v", err)
	}
	if warnings := pinWarnings(logs); len(warnings) != 1 {
		t.Fatalf("pin warnings = %q, want one for alias work", warnings)
	}
}

func TestUninstallRepointsLinkOfAliasToRemainingJDK(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks requires privileges on Windows")
	}
	setupUninstallPins(t, map[string]string{"temurin@21.0.4": "21.0.4+7", "temurin@21.0.3": "21.0.3+9"}, nil)
	if err := runAlias(t, "work", "21"); err != nil {
		t.Fatal(err)
	}
	if got := filepath.Base(aliasLinkTarget(t, "work")); got != "temurin@21.0.4" {
		t.Fatalf("jdk/work -> %s before uninstall, want temurin@21.0.4", got)
	}

	logs := uninstallLogged(t, "temurin@21.0.4")
	if got := filepath.Base(aliasLinkTarget(t, "work")); got != "temurin@21.0.3" {
		t.Fatalf("jdk/work -> %s after uninstall, want temurin@21.0.3", got)
	}
	if warnings := pinWarnings(logs); len(warnings) != 0 {
		t.Fatalf("pin warnings = %q, want none", warnings)
	}
}

func TestUninstallWarnsWhenAliasLinkCannotBeCreated(t *testing.T) {
	setupUninstallPins(t, map[string]string{"temurin@21.0.4": "21.0.4+7", "temurin@21.0.3": "21.0.3+9"},
		map[string]string{"work.alias": "21"})
	failAllSymlinks(t)
	forceGOOS(t, "windows")

	logs := uninstallLogged(t, "temurin@21.0.4")
	var aliasWarnings []string
	for _, warning := range symlinkWarnings(logs) {
		if strings.Contains(warning, filepath.Join("jdk", "work")) {
			aliasWarnings = append(aliasWarnings, warning)
		}
	}
	if len(aliasWarnings) != 1 || !strings.Contains(aliasWarnings[0], windowsSymlinkHint) {
		t.Fatalf("link warnings = %q, want one for jdk/work with the Windows hint", symlinkWarnings(logs))
	}
}

func TestUninstallWarnsWhenDefaultStopsResolving(t *testing.T) {
	warnings := uninstallWithPins(t, map[string]string{"temurin@21.0.4": "21.0.4+7"},
		map[string]string{"default-version": "21"}, "21")
	assertExists(t, "temurin@21.0.4", false)
	if len(warnings) != 1 || !strings.Contains(warnings[0], "default version 21 no longer matches an installed JDK") ||
		!strings.Contains(warnings[0], "javm default <version>") {
		t.Fatalf("warnings = %q, want one about the default", warnings)
	}
}

func TestUninstallKeepsQuietWhenDefaultStillResolves(t *testing.T) {
	warnings := uninstallWithPins(t, map[string]string{"temurin@21.0.4": "21.0.4+7", "temurin@21.0.3": "21.0.3+9"},
		map[string]string{"default-version": "21"}, "temurin@21.0.4")
	if len(warnings) != 0 {
		t.Fatalf("warnings = %q, want none while another 21 is installed", warnings)
	}
}

func TestUninstallWarnsWhenAliasStopsResolving(t *testing.T) {
	warnings := uninstallWithPins(t, map[string]string{"temurin@21.0.4": "21.0.4+7"},
		map[string]string{"work.alias": "21"}, "21")
	if len(warnings) != 1 || !strings.Contains(warnings[0], "alias work (21) no longer matches an installed JDK") ||
		!strings.Contains(warnings[0], "javm alias work <version>") || !strings.Contains(warnings[0], "javm unalias work") {
		t.Fatalf("warnings = %q, want one about alias work", warnings)
	}
}

func TestUninstallWarnsOnceForEachHolder(t *testing.T) {
	warnings := uninstallWithPins(t, map[string]string{"temurin@21.0.4": "21.0.4+7", "temurin@17.0.12": "17.0.12+7"},
		map[string]string{"default-version": "21", "work.alias": "temurin@21", "old.alias": "17"}, "21")
	if len(warnings) != 2 {
		t.Fatalf("warnings = %q, want one for the default and one for alias work", warnings)
	}
	for _, warning := range warnings {
		if strings.Contains(warning, "old") {
			t.Fatalf("warning %q names alias old, which still selects 17", warning)
		}
	}
}

func TestUninstallPinWarningIsSanitized(t *testing.T) {
	warnings := uninstallWithPins(t, map[string]string{"temurin@21.0.4": "21.0.4+7"},
		map[string]string{"work.alias": ">=21\v<22"}, "21")
	if len(warnings) != 1 || strings.ContainsAny(strings.TrimSuffix(warnings[0], "\n"), "\v\x1b") {
		t.Fatalf("warnings = %q, want one without control characters", warnings)
	}
}
