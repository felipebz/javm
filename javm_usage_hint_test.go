package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/felipebz/javm/command"
)

func TestCommandLineShapeErrorsPointToHelp(t *testing.T) {
	isolatedWorkspace(t)
	t.Setenv("HOME", t.TempDir())
	for _, tc := range []struct {
		args    []string
		message string
		path    string
	}{
		{[]string{"default"}, "accepts 1 arg(s), received 0", "javm default"},
		{[]string{"alias", "a", "b", "c"}, "accepts between 1 and 2 arg(s), received 3", "javm alias"},
		{[]string{"ls", "--bogus"}, "unknown flag: --bogus", "javm ls"},
		{[]string{"ls-remote", "--os"}, "flag needs an argument: --os", "javm ls-remote"},
		{[]string{"frobnicate"}, `unknown command "frobnicate" for "javm"`, "javm"},
		{[]string{"discover", "bogus"}, `unknown command "bogus" for "javm discover"`, "javm discover"},
		{[]string{"discover", "refresh", "x"}, `unknown command "x" for "javm discover refresh"`, "javm discover refresh"},
	} {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			stderr, err := executeRootError(t, tc.args...)
			if !errors.Is(err, command.ErrUsage) || exitCode(err) != exitUsage {
				t.Fatalf("error = %v (exit %d), want a usage error with exit 2", err, exitCode(err))
			}
			want := "Error: usage error: " + tc.message + "\nRun '" + tc.path + " --help' for usage.\n"
			if stderr != want {
				t.Fatalf("stderr = %q, want %q", stderr, want)
			}
		})
	}
}

func TestOtherErrorsHaveNoHelpHint(t *testing.T) {
	isolatedWorkspace(t)
	t.Setenv("HOME", t.TempDir())
	for _, tc := range []struct {
		args []string
		code int
	}{
		{[]string{"install", "zulu@"}, exitUsage},
		{[]string{"alias", "21", "temurin@21"}, exitUsage},
		{[]string{"which", "99"}, exitNotFound},
	} {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			stderr, err := executeRootError(t, tc.args...)
			if exitCode(err) != tc.code {
				t.Fatalf("error = %v (exit %d), want exit %d", err, exitCode(err), tc.code)
			}
			if strings.Contains(stderr, "--help' for usage") {
				t.Fatalf("stderr = %q, want no help hint", stderr)
			}
		})
	}
}
