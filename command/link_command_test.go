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
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func skipWithoutSymlinks(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks requires privileges on Windows")
	}
}

// externalJDK creates a JDK outside JAVM_HOME whose release file reports version.
func externalJDK(t *testing.T, parent, version string) string {
	t.Helper()
	root := filepath.Join(parent, "jdk-"+version)
	home := filepath.FromSlash(discovery.ExpectedJDKDir(root, runtime.GOOS))
	java := filepath.FromSlash(discovery.ExpectedJavaPath(root, runtime.GOOS))
	if err := os.MkdirAll(filepath.Dir(java), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(java, []byte("java"), 0o755); err != nil {
		t.Fatal(err)
	}
	release := "JAVA_VERSION=\"" + version + "\"\nJAVA_VENDOR=\"Test\"\nOS_ARCH=\"x86_64\"\n"
	if err := os.WriteFile(filepath.Join(home, "release"), []byte(release), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// linkTestHome isolates JAVM_HOME and limits discovery to javm's own JDKs and links.
func linkTestHome(t *testing.T) {
	t.Helper()
	t.Setenv("JAVM_HOME", t.TempDir())
	original := lsFunc
	lsFunc = func(ctx context.Context) ([]discovery.JDK, error) {
		return discovery.NewJavmSource().Discover(ctx)
	}
	t.Cleanup(func() { lsFunc = original })
}

func runLinkCommand(t *testing.T, cmd *cobra.Command, args ...string) (string, error) {
	t.Helper()
	var logs bytes.Buffer
	logger := log.New()
	logger.SetOutput(&logs)
	logger.SetFormatter(&log.TextFormatter{DisableQuote: true, DisableTimestamp: true})
	ctx := WithRuntime(context.Background(), Runtime{Logger: logger, Err: &bytes.Buffer{}})
	cmd.SetArgs(args)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SilenceUsage = true
	err := cmd.ExecuteContext(ctx)
	return logs.String(), err
}

func linkPointsAt(t *testing.T, identifier, dir string) {
	t.Helper()
	target, err := os.Readlink(filepath.Join(cfg.Dir(), "jdk", identifier))
	if err != nil || target != dir {
		t.Fatalf("jdk/%s -> %q, %v, want %q", identifier, target, err, dir)
	}
}

func TestLinkDerivesIdentifierFromJDK(t *testing.T) {
	skipWithoutSymlinks(t)
	linkTestHome(t)
	dir := externalJDK(t, t.TempDir(), "21.0.4")

	logs, err := runLinkCommand(t, NewLinkCommand(), dir)
	if err != nil {
		t.Fatal(err)
	}
	linkPointsAt(t, "system@21.0.4", dir)
	if !strings.Contains(logs, "system@21.0.4 -> "+dir) {
		t.Fatalf("logs = %q, want what was created", logs)
	}
	jdk, err := resolveJDKContext(context.Background(), "system@21.0.4")
	if err != nil || jdk.Path != filepath.Join(cfg.Dir(), "jdk", "system@21.0.4") {
		t.Fatalf("resolve system@21.0.4 = %+v, %v", jdk, err)
	}
}

func TestLinkNormalizesLegacyJavaVersion(t *testing.T) {
	skipWithoutSymlinks(t)
	linkTestHome(t)
	dir := externalJDK(t, t.TempDir(), "1.8.0_402")

	if _, err := runLinkCommand(t, NewLinkCommand(), dir); err != nil {
		t.Fatal(err)
	}
	linkPointsAt(t, "system@8.0.402", dir)
}

func TestLinkNameOverridesVersion(t *testing.T) {
	skipWithoutSymlinks(t)
	for _, name := range []string{"21-custom", "system@21-custom"} {
		t.Run(name, func(t *testing.T) {
			linkTestHome(t)
			dir := externalJDK(t, t.TempDir(), "21.0.4")
			if _, err := runLinkCommand(t, NewLinkCommand(), dir, "--name", name); err != nil {
				t.Fatal(err)
			}
			linkPointsAt(t, "system@21-custom", dir)
		})
	}
}

func TestLinkSamePathIsNoOpAndOtherPathIsRefused(t *testing.T) {
	skipWithoutSymlinks(t)
	linkTestHome(t)
	parent := t.TempDir()
	dir := externalJDK(t, parent, "21.0.4")
	if _, err := runLinkCommand(t, NewLinkCommand(), dir); err != nil {
		t.Fatal(err)
	}
	if _, err := runLinkCommand(t, NewLinkCommand(), dir); err != nil {
		t.Fatalf("linking the same path again error = %v, want success", err)
	}

	other := externalJDK(t, t.TempDir(), "21.0.4")
	_, err := runLinkCommand(t, NewLinkCommand(), other)
	if !errors.Is(err, ErrUsage) || !strings.Contains(err.Error(), dir) || !strings.Contains(err.Error(), "--name") ||
		!strings.Contains(err.Error(), "javm unlink system@21.0.4") || strings.Contains(err.Error(), "file exists") {
		t.Fatalf("link of another path error = %v, want a usage error naming %s", err, dir)
	}
	linkPointsAt(t, "system@21.0.4", dir)
}

func TestLinkAcceptsDeprecatedForm(t *testing.T) {
	skipWithoutSymlinks(t)
	linkTestHome(t)
	dir := externalJDK(t, t.TempDir(), "1.8.0_20")

	logs, err := runLinkCommand(t, NewLinkCommand(), "system@1.8.20", dir)
	if err != nil {
		t.Fatal(err)
	}
	linkPointsAt(t, "system@1.8.20", dir)
	if strings.Count(logs, "deprecated") != 1 || !strings.Contains(logs, "--name 1.8.20") {
		t.Fatalf("logs = %q, want one deprecation warning showing the new form", logs)
	}
}

func TestLinkRejectsRemovedForms(t *testing.T) {
	linkTestHome(t)
	if _, err := runLinkCommand(t, NewLinkCommand()); !errors.Is(err, ErrUsage) || !strings.Contains(err.Error(), "a JDK path is required") {
		t.Fatalf("link error = %v, want a usage error asking for a path", err)
	}
	if _, err := runLinkCommand(t, NewLinkCommand(), "system@21.0.4"); !errors.Is(err, ErrUsage) || !strings.Contains(err.Error(), "javm which system@21.0.4") {
		t.Fatalf("link system@21.0.4 error = %v, want a usage error suggesting which", err)
	}
}

func TestUnlinkMatchesIdentifierVersionAndPath(t *testing.T) {
	skipWithoutSymlinks(t)
	for _, arg := range []string{"system@21.0.4", "21.0.4", "<dir>"} {
		t.Run(arg, func(t *testing.T) {
			linkTestHome(t)
			dir := externalJDK(t, t.TempDir(), "21.0.4")
			if _, err := runLinkCommand(t, NewLinkCommand(), dir); err != nil {
				t.Fatal(err)
			}
			if arg == "<dir>" {
				arg = dir
			}
			logs, err := runLinkCommand(t, NewUnlinkCommand(), arg)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := os.Lstat(filepath.Join(cfg.Dir(), "jdk", "system@21.0.4")); !os.IsNotExist(err) {
				t.Fatalf("link still exists: %v", err)
			}
			if !strings.Contains(logs, "Removed link system@21.0.4") {
				t.Fatalf("logs = %q, want the removal", logs)
			}
		})
	}
}

func TestUnlinkRefusesAmbiguousPathAndUnknownNames(t *testing.T) {
	skipWithoutSymlinks(t)
	linkTestHome(t)
	writeManagedJDK(t, "temurin@21.0.4", "21.0.4+7", "")
	dir := externalJDK(t, t.TempDir(), "21.0.4")
	for _, name := range []string{"21-a", "21-b"} {
		if _, err := runLinkCommand(t, NewLinkCommand(), dir, "--name", name); err != nil {
			t.Fatal(err)
		}
	}

	_, err := runLinkCommand(t, NewUnlinkCommand(), dir)
	if !errors.Is(err, ErrUsage) || !strings.Contains(err.Error(), "system@21-a") || !strings.Contains(err.Error(), "system@21-b") {
		t.Fatalf("unlink <dir> error = %v, want a usage error listing both links", err)
	}
	linkPointsAt(t, "system@21-a", dir)
	linkPointsAt(t, "system@21-b", dir)

	if _, err := runLinkCommand(t, NewUnlinkCommand(), "99"); !errors.Is(err, ErrNotFound) || !strings.Contains(err.Error(), "javm ls") {
		t.Fatalf("unlink 99 error = %v, want not found suggesting 'javm ls'", err)
	}
	if _, err := runLinkCommand(t, NewUnlinkCommand(), "temurin@21.0.4"); !errors.Is(err, ErrUsage) || !strings.Contains(err.Error(), "javm uninstall temurin@21.0.4") {
		t.Fatalf("unlink temurin@21.0.4 error = %v, want a usage error pointing at uninstall", err)
	}
	assertExists(t, "temurin@21.0.4", true)
}

func TestUninstallPointsAtUnlinkForSystemLink(t *testing.T) {
	skipWithoutSymlinks(t)
	linkTestHome(t)
	dir := externalJDK(t, t.TempDir(), "21.0.4")
	if _, err := runLinkCommand(t, NewLinkCommand(), dir); err != nil {
		t.Fatal(err)
	}
	_, err := runLinkCommand(t, NewUninstallCommand(), "21")
	if !errors.Is(err, ErrUsage) || !strings.Contains(err.Error(), "javm unlink system@21.0.4") {
		t.Fatalf("uninstall 21 error = %v, want a usage error pointing at unlink", err)
	}
	linkPointsAt(t, "system@21.0.4", dir)
}

func TestLinkFailsWithWindowsHint(t *testing.T) {
	linkTestHome(t)
	dir := externalJDK(t, t.TempDir(), "21.0.4")
	failAllSymlinks(t)
	forceGOOS(t, "windows")
	if _, err := runLinkCommand(t, NewLinkCommand(), dir); err == nil || !strings.Contains(err.Error(), windowsSymlinkHint) {
		t.Fatalf("link error = %v, want the Windows hint", err)
	}
}

func TestLinkMessagesAreSanitized(t *testing.T) {
	skipWithoutSymlinks(t)
	linkTestHome(t)
	dir := externalJDK(t, filepath.Join(t.TempDir(), "jdks\x1b[31m"), "21.0.4")
	logs, err := runLinkCommand(t, NewLinkCommand(), dir)
	if err != nil {
		t.Fatal(err)
	}
	other := externalJDK(t, t.TempDir(), "21.0.4")
	_, err = runLinkCommand(t, NewLinkCommand(), other)
	if err == nil || strings.ContainsRune(err.Error(), '\x1b') || strings.ContainsRune(logs, '\x1b') {
		t.Fatalf("logs = %q, error = %q, want no control characters", logs, err)
	}
}
