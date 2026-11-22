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

// skipIfAncestorHasJavaVersion skips tests that expect no .java-version to be
// found, which a file in an ancestor of the temp directory would break.
func skipIfAncestorHasJavaVersion(t *testing.T, dir string) {
	t.Helper()
	for parent := filepath.Dir(dir); ; parent = filepath.Dir(parent) {
		if _, err := os.Lstat(filepath.Join(parent, ".java-version")); err == nil {
			t.Skipf("%s has a .java-version, so nothing can be missing below it", parent)
		}
		if filepath.Dir(parent) == parent {
			return
		}
	}
}

func TestJavaVersionSelectorReportsNotFound(t *testing.T) {
	dir := t.TempDir()
	skipIfAncestorHasJavaVersion(t, dir)
	t.Chdir(dir)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	_, err = javaVersionSelector()
	want := "no version given and no .java-version found in " + cwd + " or its parent directories"
	if err == nil || err.Error() != want {
		t.Fatalf("javaVersionSelector() error = %v, want %q", err, want)
	}
}

func TestJavaVersionSelectorSearchesParents(t *testing.T) {
	tests := []struct {
		name    string
		files   map[string]string // relative to the project root; a trailing / makes a directory
		want    string
		wantErr string // a path relative to the root that the error must name, plus its text
		errText string
	}{
		{name: "parent", files: map[string]string{"a/.java-version": " temurin@21\n"}, want: "temurin@21"},
		{name: "two levels up", files: map[string]string{".java-version": "zulu@17\n"}, want: "zulu@17"},
		{name: "nearest wins", files: map[string]string{".java-version": "zulu@17", "a/.java-version": "temurin@21"}, want: "temurin@21"},
		{
			name:    "empty nearest file does not fall back",
			files:   map[string]string{"a/b/.java-version": "", ".java-version": "zulu@17"},
			wantErr: "a/b/.java-version", errText: "is empty",
		},
		{
			name:    "unreadable nearest entry does not fall back",
			files:   map[string]string{"a/b/.java-version/": "", ".java-version": "zulu@17"},
			wantErr: "a/b/.java-version", errText: "read ",
		},
		{
			name:    "empty file in the parent",
			files:   map[string]string{"a/.java-version": " \n"},
			wantErr: "a/.java-version", errText: "is empty",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			workdir := filepath.Join(root, "a", "b")
			if err := os.MkdirAll(workdir, 0o755); err != nil {
				t.Fatal(err)
			}
			for name, content := range tt.files {
				path := filepath.Join(root, filepath.FromSlash(strings.TrimSuffix(name, "/")))
				var err error
				if strings.HasSuffix(name, "/") {
					err = os.Mkdir(path, 0o755)
				} else {
					err = os.WriteFile(path, []byte(content), 0o600)
				}
				if err != nil {
					t.Fatal(err)
				}
			}
			t.Chdir(workdir)

			got, err := javaVersionSelector()
			if tt.wantErr == "" {
				if err != nil || got != tt.want {
					t.Fatalf("javaVersionSelector() = %q, %v, want %q", got, err, tt.want)
				}
				return
			}
			wantPath := filepath.Join(root, filepath.FromSlash(tt.wantErr))
			if err == nil || !strings.Contains(err.Error(), wantPath) || !strings.Contains(err.Error(), tt.errText) {
				t.Fatalf("javaVersionSelector() error = %v, want %q naming %s", err, tt.errText, wantPath)
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
	// An empty file names the path in the error without depending on what
	// the ancestors of the temp directory contain.
	writeJavaVersion(t, "")
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
