package command

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
)

func TestJavaVersionSelector(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T)
		want    string
		wantErr string
	}{
		{
			name: "trimmed selector",
			setup: func(t *testing.T) {
				writeJavaVersion(t, "  temurin@21 \r\n")
			},
			want: "temurin@21",
		},
		{
			name:    "missing file",
			setup:   func(*testing.T) {},
			wantErr: "no version given and no .java-version found in ",
		},
		{
			name:    "empty file",
			setup:   func(t *testing.T) { writeJavaVersion(t, "") },
			wantErr: ".java-version is empty",
		},
		{
			name:    "whitespace only",
			setup:   func(t *testing.T) { writeJavaVersion(t, " \n\t\n") },
			wantErr: ".java-version is empty",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			tt.setup(t)
			got, err := javaVersionSelector()
			if tt.wantErr == "" {
				if err != nil || got != tt.want {
					t.Fatalf("javaVersionSelector() = %q, %v, want %q", got, err, tt.want)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("javaVersionSelector() error = %v, want %q", err, tt.wantErr)
			}
			if got != "" {
				t.Fatalf("javaVersionSelector() = %q with an error", got)
			}
		})
	}
}

func TestJavaVersionSelectorWrapsReadError(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.Mkdir(".java-version", 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := javaVersionSelector()
	if err == nil || !strings.Contains(err.Error(), ".java-version") || !strings.HasPrefix(err.Error(), "read ") {
		t.Fatalf("javaVersionSelector() error = %v, want a read error naming the file", err)
	}
	if runtime.GOOS == "linux" && !errors.Is(err, syscall.EISDIR) {
		t.Fatalf("cause is not reachable with errors.Is: %v", err)
	}
}

func TestJavaVersionSelectorSanitizesDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not allow control characters in file names")
	}
	dir := filepath.Join(t.TempDir(), "project\x1b[31mred")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	_, err := javaVersionSelector()
	if err == nil || strings.ContainsRune(err.Error(), '\x1b') {
		t.Fatalf("javaVersionSelector() error = %q, want the directory sanitized", err)
	}
}

func writeJavaVersion(t *testing.T, content string) {
	t.Helper()
	if err := os.WriteFile(".java-version", []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
