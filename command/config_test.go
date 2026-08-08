package command

import (
	"bytes"
	"strings"
	"testing"
)

func TestConfigUsesCobraStreams(t *testing.T) {
	t.Setenv("JAVM_HOME", t.TempDir())

	t.Run("data goes to stdout", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		cmd := NewConfigCommand()
		cmd.SetOut(&stdout)
		cmd.SetErr(&stderr)
		cmd.SetArgs([]string{"get", "java.default_distribution"})

		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		if got := stdout.String(); got != "temurin\n" {
			t.Fatalf("stdout = %q, want %q", got, "temurin\n")
		}
		if stderr.Len() != 0 {
			t.Fatalf("stderr = %q, want empty", stderr.String())
		}
	})

	t.Run("errors do not contaminate stdout", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		cmd := NewConfigCommand()
		cmd.SetOut(&stdout)
		cmd.SetErr(&stderr)
		cmd.SetArgs([]string{"get", "unknown"})

		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "unknown key") {
			t.Fatalf("expected unknown key error, got %v", err)
		}
		if stdout.Len() != 0 {
			t.Fatalf("stdout = %q, want empty", stdout.String())
		}
		if !strings.Contains(stderr.String(), "unknown key") {
			t.Fatalf("stderr = %q, want diagnostic", stderr.String())
		}
	})
}
