package main

import (
	"strings"
	"testing"
)

func TestAliasCommandsReportUnsetAlias(t *testing.T) {
	for _, command := range []string{"alias", "unalias"} {
		t.Run(command, func(t *testing.T) {
			isolatedWorkspace(t)
			stdout, stderr, code := executeRoot(t, command, "nope")
			if code != exitNotFound || stdout != "" {
				t.Fatalf("exit code = %d, stdout = %q, stderr = %q, want exit %d and no output", code, stdout, stderr, exitNotFound)
			}
			if !strings.Contains(stderr, `alias "nope" is not set`) || strings.Contains(stderr, ".alias") || strings.Contains(stderr, "no such file") {
				t.Fatalf("stderr = %q, want only the alias name, not the file", stderr)
			}
		})
	}
}

func TestAliasLifecycle(t *testing.T) {
	_, home := isolatedWorkspace(t)
	writeFakeManagedJDK(t, home)

	if _, stderr, code := executeRoot(t, "alias", "work", "temurin@21"); code != exitSuccess {
		t.Fatalf("alias work temurin@21: exit code = %d, stderr = %q", code, stderr)
	}
	if stdout, stderr, code := executeRoot(t, "alias", "work"); code != exitSuccess || stdout != "temurin@21\n" {
		t.Fatalf("alias work: exit code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
	}
	if _, stderr, code := executeRoot(t, "unalias", "work"); code != exitSuccess {
		t.Fatalf("unalias work: exit code = %d, stderr = %q", code, stderr)
	}
	if _, stderr, code := executeRoot(t, "alias", "work"); code != exitNotFound {
		t.Fatalf("alias work after unalias: exit code = %d, stderr = %q, want %d", code, stderr, exitNotFound)
	}
}

func TestAliasCommandsRejectInvalidNames(t *testing.T) {
	for _, command := range []string{"alias", "unalias"} {
		t.Run(command, func(t *testing.T) {
			isolatedWorkspace(t)
			if _, stderr, code := executeRoot(t, command, "../x"); code != exitUsage {
				t.Fatalf("exit code = %d, stderr = %q, want %d", code, stderr, exitUsage)
			}
		})
	}
}
