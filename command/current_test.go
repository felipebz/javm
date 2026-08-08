package command

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/felipebz/javm/cfg"
)

func TestCurrent(t *testing.T) {
	previousLookPath := lookPath
	defer func() { lookPath = previousLookPath }()
	lookPath = func(string) (string, error) {
		return filepath.Join(cfg.Dir(), "jdk", "1.8.0", "Contents", "Home", "bin", "java"), nil
	}
	actual := current()
	expected := "1.8.0"
	if actual != expected {
		t.Fatalf("actual: %v != expected: %v", actual, expected)
	}
}

func TestCurrentUsesCobraOutput(t *testing.T) {
	home := t.TempDir()
	t.Setenv("JAVM_HOME", home)
	originalLookPath := lookPath
	lookPath = func(string) (string, error) {
		return filepath.Join(home, "jdk", "temurin@21", "bin", "java"), nil
	}
	t.Cleanup(func() { lookPath = originalLookPath })

	var stdout bytes.Buffer
	cmd := NewCurrentCommand()
	cmd.SetOut(&stdout)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	want := "temurin@21\n"
	if got := stdout.String(); got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}
