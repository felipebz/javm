package main

import (
	"strings"
	"testing"
)

func TestUnknownSubcommandIsUsageError(t *testing.T) {
	for _, args := range [][]string{{"fooo"}, {"config", "foo"}, {"config", "lst"}, {"discover", "list"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			isolatedWorkspace(t)
			stdout, stderr, code := executeRoot(t, args...)
			path := "javm"
			if len(args) > 1 {
				path += " " + args[0]
			}
			want := `Error: usage error: unknown command "` + args[len(args)-1] + `" for "` + path + `"`
			if code != exitUsage || strings.Contains(stdout, "Usage:") || strings.Count(stderr, want) != 1 {
				t.Fatalf("exit %d, stdout %q, stderr %q, want exit %d and one %q", code, stdout, stderr, exitUsage, want)
			}
			if strings.Contains(stderr, "Did you mean") {
				t.Fatalf("stderr %q, want no suggestions, as for the root command", stderr)
			}
		})
	}
}

func TestParentCommandsStillShowHelp(t *testing.T) {
	for _, args := range [][]string{{"config"}, {"discover"}, {"config", "--help"}, {"discover", "-h"}, {"discover", "refresh", "--help"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			isolatedWorkspace(t)
			stdout, stderr, code := executeRoot(t, args...)
			if code != exitSuccess || !strings.Contains(stdout, "Usage:") || stderr != "" {
				t.Fatalf("exit %d, stdout %q, stderr %q, want the help and exit 0", code, stdout, stderr)
			}
		})
	}
}

func TestConfigListStillRuns(t *testing.T) {
	isolatedWorkspace(t)
	stdout, stderr, code := executeRoot(t, "config", "list")
	if code != exitSuccess || strings.Contains(stdout, "Usage:") || stderr != "" {
		t.Fatalf("exit %d, stdout %q, stderr %q, want the configuration listing", code, stdout, stderr)
	}
}
