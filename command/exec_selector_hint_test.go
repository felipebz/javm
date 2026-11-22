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
)

// execWorkspace runs in an empty directory with one managed temurin@21.0.4,
// and writes .java-version there when javaVersion is not empty.
func execWorkspace(t *testing.T, javaVersion string) {
	t.Helper()
	t.Setenv("JAVM_HOME", t.TempDir())
	managedDiscoveryOnly(t)
	writeManagedJDK(t, "temurin@21.0.4", "21.0.4+7", "")
	dir := t.TempDir()
	t.Chdir(dir)
	if javaVersion != "" {
		if err := os.WriteFile(filepath.Join(dir, ".java-version"), []byte(javaVersion), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func runExecCommand(t *testing.T, args ...string) error {
	t.Helper()
	cmd := NewExecCommand()
	cmd.SetArgs(args)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SilenceUsage = true
	return cmd.ExecuteContext(context.Background())
}

func TestExecSuggestsJDKFlagWithoutJavaVersion(t *testing.T) {
	execWorkspace(t, "")
	if err := os.WriteFile(filepath.Join(cfg.Dir(), "work.alias"), []byte("21"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"21", "--", "java", "-version"}, "javm exec --jdk 21 -- java -version"},
		{[]string{"temurin@21", "java"}, "javm exec --jdk temurin@21 -- java"},
		{[]string{"work", "java"}, "javm exec --jdk work -- java"},
		{[]string{"21", "--", "java", "-Dname=a b", `say "hi"`}, `javm exec --jdk 21 -- java "-Dname=a b" 'say "hi"'`},
	} {
		err := runExecCommand(t, tc.args...)
		if !errors.Is(err, ErrUsage) || !strings.Contains(err.Error(), "no .java-version found") ||
			!strings.Contains(err.Error(), "looks like a JDK selector") || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("exec %q error = %v, want the original message and a hint with %s", tc.args, err, tc.want)
		}
	}
}

func TestExecWithoutSelectorWordHasNoHint(t *testing.T) {
	execWorkspace(t, "")
	err := runExecCommand(t, "java", "-version")
	if !errors.Is(err, ErrUsage) || strings.Contains(err.Error(), "looks like a JDK selector") {
		t.Fatalf("exec java error = %v, want the original message without a hint", err)
	}
}

func TestExecSuggestsJDKFlagWhenSelectorWordIsNotACommand(t *testing.T) {
	execWorkspace(t, "21")
	t.Setenv("PATH", t.TempDir())
	err := runExecCommand(t, "21", "--", "java", "-version")
	if !errors.Is(err, ErrNotFound) || !strings.Contains(err.Error(), `command "21" was not found`) ||
		!strings.Contains(err.Error(), "javm exec --jdk 21 -- java -version") {
		t.Fatalf("exec error = %v, want the not found error with the hint", err)
	}
}

func TestExecRunsProgramNamedLikeSelector(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a shell script")
	}
	execWorkspace(t, "21")
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "21"), []byte("#!/bin/sh\nexit 7\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	err := runExecCommand(t, "21", "--", "java")
	var childErr *childExitError
	if !errors.As(err, &childErr) || childErr.ExitCode() != 7 || strings.Contains(err.Error(), "looks like a JDK selector") {
		t.Fatalf("exec error = %v, want the script's exit status 7 without a hint", err)
	}
}
