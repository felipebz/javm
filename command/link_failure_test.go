package command

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"

	"github.com/felipebz/javm/cfg"
	"github.com/felipebz/javm/discoapi"
	"github.com/felipebz/javm/discovery"
)

const windowsSymlinkHint = "creating symlinks on Windows requires Developer Mode or an elevated prompt"

// failSymlinks makes symlink creation fail with ERROR_PRIVILEGE_NOT_HELD for
// the link names fail accepts, and creates the others normally.
func failSymlinks(t *testing.T, fail func(name string) bool) {
	t.Helper()
	original := createSymlink
	createSymlink = func(oldname, newname string) error {
		if fail(filepath.Base(newname)) {
			return &os.LinkError{Op: "symlink", Old: oldname, New: newname, Err: errPrivilegeNotHeld}
		}
		return os.Symlink(oldname, newname)
	}
	t.Cleanup(func() { createSymlink = original })
}

func failAllSymlinks(t *testing.T) {
	failSymlinks(t, func(string) bool { return true })
}

func forceGOOS(t *testing.T, goos string) {
	t.Helper()
	original := hostGOOS
	hostGOOS = goos
	t.Cleanup(func() { hostGOOS = original })
}

func symlinkWarnings(logs *bytes.Buffer) []string {
	var warnings []string
	for line := range strings.Lines(logs.String()) {
		if strings.Contains(line, "level=warning") && strings.Contains(line, "create symlink") {
			warnings = append(warnings, line)
		}
	}
	return warnings
}

func TestInstallSucceedsWhenConvenienceLinksFail(t *testing.T) {
	t.Setenv("JAVM_HOME", t.TempDir())
	failAllSymlinks(t)
	forceGOOS(t, "windows")
	if err := setAlias("default", "temurin@17"); err != nil {
		t.Fatal(err)
	}
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
	jdks, err := LsContext(ctx, true)
	if err != nil || len(jdks) != 1 || jdks[0].Identifier != "temurin@17.0.12" {
		t.Fatalf("LsContext() = %+v, %v, want the installed JDK", jdks, err)
	}
	warnings := symlinkWarnings(logs)
	if len(warnings) != 2 {
		t.Fatalf("got %d link warnings, want one for temurin@17.0 and one for default:\n%s", len(warnings), logs.String())
	}
	for _, warning := range warnings {
		if !strings.Contains(warning, windowsSymlinkHint) {
			t.Fatalf("warning %q lacks the Windows hint", warning)
		}
	}
}

func TestUpdateSucceedsWhenConvenienceLinksFail(t *testing.T) {
	client := temurinUpdateFixture(t)
	failAllSymlinks(t)
	ctx, logs := contextWithLogBuffer(context.Background())

	if err := runUpdate(ctx, client, &bytes.Buffer{}, "temurin@25", updateOptions{}); err != nil {
		t.Fatalf("runUpdate() error = %v", err)
	}
	assertExists(t, "temurin@25.0.5", true)
	if warnings := symlinkWarnings(logs); len(warnings) != 1 || !strings.Contains(warnings[0], "temurin@25.0") {
		t.Fatalf("link warnings = %q, want one for temurin@25.0", warnings)
	}
}

func TestUninstallSucceedsWhenConvenienceLinksFail(t *testing.T) {
	t.Setenv("JAVM_HOME", t.TempDir())
	writeManagedJDK(t, "temurin@21.0.4", "21.0.4+7", "")
	writeManagedJDK(t, "temurin@21.0.3", "21.0.3+9", "")
	failAllSymlinks(t)
	ctx, logs := contextWithLogBuffer(context.Background())

	cmd := NewUninstallCommand()
	cmd.SetArgs([]string{"temurin@21.0.4"})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	if err := cmd.ExecuteContext(ctx); err != nil {
		t.Fatalf("uninstall error = %v", err)
	}
	assertExists(t, "temurin@21.0.4", false)
	if warnings := symlinkWarnings(logs); len(warnings) != 1 {
		t.Fatalf("link warnings = %q, want one for temurin@21.0", warnings)
	}
}

func TestLinkLatestContinuesAfterOneLinkFails(t *testing.T) {
	for _, failing := range []string{"temurin@17.0", "temurin@21.0"} {
		t.Run(failing, func(t *testing.T) {
			t.Setenv("JAVM_HOME", t.TempDir())
			writeManagedJDK(t, "temurin@17.0.12", "17.0.12+7", "")
			writeManagedJDK(t, "temurin@21.0.4", "21.0.4+7", "")
			failSymlinks(t, func(name string) bool { return name == failing })

			err := linkLatest(context.Background())
			var refreshErr *linkRefreshError
			if !errors.As(err, &refreshErr) || !errors.Is(err, errPrivilegeNotHeld) {
				t.Fatalf("linkLatest() error = %v, want a *linkRefreshError wrapping the symlink failure", err)
			}
			for _, name := range []string{"temurin@17.0", "temurin@21.0"} {
				_, statErr := os.Lstat(filepath.Join(cfg.Dir(), "jdk", name))
				if created := statErr == nil; created == (name == failing) {
					t.Fatalf("%s created = %v, want only %s to be missing", name, created, failing)
				}
			}
		})
	}
}

func TestLinkRefreshStillFailsWithHint(t *testing.T) {
	for _, goos := range []string{"windows", "linux"} {
		t.Run(goos, func(t *testing.T) {
			t.Setenv("JAVM_HOME", t.TempDir())
			writeManagedJDK(t, "temurin@21.0.4", "21.0.4+7", "")
			failAllSymlinks(t)
			forceGOOS(t, goos)

			cmd := NewLinkCommand()
			// nil would make Cobra read os.Args, which holds the test flags.
			cmd.SetArgs([]string{})
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})
			cmd.SilenceUsage = true
			err := cmd.ExecuteContext(context.Background())
			if err == nil {
				t.Fatal("javm link succeeded although the link could not be created")
			}
			if got := strings.Contains(err.Error(), windowsSymlinkHint); got != (goos == "windows") {
				t.Fatalf("error %q contains the hint = %v, want %v", err, got, goos == "windows")
			}
		})
	}
}

func TestSymlinkCommandsFailWithHint(t *testing.T) {
	t.Run("alias", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("JAVM_HOME", home)
		root := filepath.Join(home, "jdk", "temurin@21.0.4")
		writeManagedJDK(t, "temurin@21.0.4", "21.0.4+7", "")
		cleanup := setupMockLs()
		t.Cleanup(cleanup)
		mockLsResult = []discovery.JDK{{Identifier: "temurin@21.0.4", Source: "javm", Path: root}}
		failAllSymlinks(t)
		forceGOOS(t, "windows")

		cmd := NewAliasCommand()
		cmd.SetArgs([]string{"work", "temurin@21"})
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		cmd.SilenceUsage = true
		if err := cmd.ExecuteContext(context.Background()); err == nil || !strings.Contains(err.Error(), windowsSymlinkHint) {
			t.Fatalf("alias error = %v, want a failure with the Windows hint", err)
		}
	})

	t.Run("link system JDK", func(t *testing.T) {
		t.Setenv("JAVM_HOME", t.TempDir())
		jdk := filepath.Join(t.TempDir(), "external-jdk")
		makeFakeJDK(t, jdk, runtime.GOOS == "darwin")
		failAllSymlinks(t)
		forceGOOS(t, "windows")

		cmd := NewLinkCommand()
		cmd.SetArgs([]string{"system@21.0.1", jdk})
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		cmd.SilenceUsage = true
		if err := cmd.ExecuteContext(context.Background()); err == nil || !strings.Contains(err.Error(), windowsSymlinkHint) {
			t.Fatalf("link error = %v, want a failure with the Windows hint", err)
		}
	})
}

func TestInstallStillFailsOnOtherLinkErrors(t *testing.T) {
	t.Setenv("JAVM_HOME", t.TempDir())
	original := readDir
	readDir = func(string) ([]os.DirEntry, error) { return nil, fs.ErrPermission }
	t.Cleanup(func() { readDir = original })
	client := &updateClient{
		packages: []discoapi.Package{{Id: "17", JavaVersion: "17.0.12+7", Distribution: "temurin", DistributionVersion: "17.0.12"}},
		archives: map[string]string{"17": updateArchive(t, "17.0.12+7", "new")},
	}

	cmd := NewInstallCommand(client)
	cmd.SetArgs([]string{"temurin@17"})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SilenceUsage = true
	if err := cmd.ExecuteContext(context.Background()); !errors.Is(err, fs.ErrPermission) {
		t.Fatalf("install error = %v, want the jdk/ read error", err)
	}
}

func TestSymlinkHint(t *testing.T) {
	tests := []struct {
		name string
		goos string
		err  error
		want bool
	}{
		{name: "privilege error on Windows", goos: "windows", err: &os.LinkError{Op: "symlink", Err: errPrivilegeNotHeld}, want: true},
		{name: "wrapped privilege error on Windows", goos: "windows", err: &linkRefreshError{failures: []error{&os.LinkError{Op: "symlink", Err: errPrivilegeNotHeld}}}, want: true},
		{name: "other errno on Windows", goos: "windows", err: &os.LinkError{Op: "symlink", Err: syscall.Errno(5)}},
		{name: "privilege errno elsewhere", goos: "linux", err: &os.LinkError{Op: "symlink", Err: errPrivilegeNotHeld}},
		{name: "nil", goos: "windows"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			forceGOOS(t, tt.goos)
			if got := symlinkHint(tt.err); (got != "") != tt.want || (tt.want && !strings.Contains(got, windowsSymlinkHint)) {
				t.Fatalf("symlinkHint() = %q, want hint %v", got, tt.want)
			}
		})
	}
}
