package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestRootExecPassesChildExitStatusSilently(t *testing.T) {
	probe := createRootExecFixture(t)
	t.Setenv("JAVM_TEST_EXIT", "7")

	stdout, stderr, code := executeRoot(t, "exec", "--jdk", "21", filepath.Base(probe))
	if code != 7 || stderr != "" || !strings.Contains(stdout, "LABEL=") {
		t.Fatalf("exit %d, stdout %q, stderr %q, want the child's exit 7 and nothing from javm", code, stdout, stderr)
	}
}

func TestRootExecReportsItsOwnFailures(t *testing.T) {
	probe := createRootExecFixture(t)
	for name, tt := range map[string]struct {
		args []string
		want string
	}{
		"unknown selector":  {args: []string{"exec", "--jdk", "99", filepath.Base(probe)}, want: "99 isn't installed"},
		"command not found": {args: []string{"exec", "--jdk", "21", "definitely-not-installed"}, want: `command "definitely-not-installed" was not found`},
	} {
		t.Run(name, func(t *testing.T) {
			_, stderr, code := executeRoot(t, tt.args...)
			if code != exitNotFound || strings.Count(stderr, tt.want) != 1 {
				t.Fatalf("exit %d, stderr %q, want exit %d and %q once", code, stderr, exitNotFound, tt.want)
			}
		})
	}
}

func TestRootExecAcceptsRootFlagsBeforeTheCommand(t *testing.T) {
	probe := createRootExecFixture(t)
	for _, args := range [][]string{
		{"--quiet", "exec", "--jdk=21", filepath.Base(probe), "--quiet"},
		{"exec", "--quiet", "--jdk=21", "--", filepath.Base(probe), "--quiet"},
	} {
		stdout, stderr, code := executeRoot(t, args...)
		if code != exitSuccess || !strings.Contains(stdout, `ARGS=["--quiet"]`) {
			t.Fatalf("%q: exit %d, stdout %q, stderr %q, want the child to get only its own --quiet", args, code, stdout, stderr)
		}
	}
}
