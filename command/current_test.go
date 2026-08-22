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

	tests := []struct {
		name       string
		javaSubdir string
		want       string
	}{
		{name: "managed bin directory", javaSubdir: filepath.Join("1.8.0", "bin"), want: "1.8.0"},
		{name: "macOS Contents/Home layout", javaSubdir: filepath.Join("1.8.0", "Contents", "Home", "bin"), want: "1.8.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lookPath = func(string) (string, error) {
				return filepath.Join(cfg.Dir(), "jdk", tt.javaSubdir, "java"), nil
			}
			if got := current(); got != tt.want {
				t.Fatalf("current() = %q, want %q", got, tt.want)
			}
		})
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
