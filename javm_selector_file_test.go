package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Editors and `echo` end hand-edited files with a newline.
func TestUseDefaultAcceptsTrailingNewline(t *testing.T) {
	for name, content := range map[string]string{"LF": "21\n", "CRLF": "21\r\n", "blank lines": "21 \n\n"} {
		t.Run(name, func(t *testing.T) {
			_, home := isolatedWorkspace(t)
			jdk := writeFakeManagedJDK(t, home)
			if err := os.WriteFile(filepath.Join(home, "default-version"), []byte(content), 0o600); err != nil {
				t.Fatal(err)
			}
			envFile := filepath.Join(t.TempDir(), "environment")

			_, stderr, code := executeRoot(t, "use", "--fd3", envFile, "--default")
			data, err := os.ReadFile(envFile)
			if code != exitSuccess || stderr != "" || err != nil || !strings.Contains(string(data), filepath.Base(jdk)) {
				t.Fatalf("exit code = %d, records = %q, %v, stderr = %q, want the default activated quietly", code, data, err, stderr)
			}
		})
	}
}

func TestUseDefaultRejectsMultilineDefault(t *testing.T) {
	for name, content := range map[string]string{"command": "21\necho compromised", "two lines": "21\n17\n", "NUL": "21\x00"} {
		t.Run(name, func(t *testing.T) {
			_, home := isolatedWorkspace(t)
			writeFakeManagedJDK(t, home)
			if err := os.WriteFile(filepath.Join(home, "default-version"), []byte(content), 0o600); err != nil {
				t.Fatal(err)
			}
			envFile := filepath.Join(t.TempDir(), "environment")

			stdout, stderr, code := executeRoot(t, "use", "--fd3", envFile, "--default")
			if code != exitSuccess || stdout != "" {
				t.Fatalf("exit code = %d, stdout = %q, stderr = %q, want a quiet success", code, stdout, stderr)
			}
			if data, err := os.ReadFile(envFile); err == nil && len(data) != 0 {
				t.Fatalf("use --default wrote records: %q", data)
			}
			want := filepath.Join(home, "default-version") + ": default version selector contains a forbidden control character; run 'javm default <version>' to replace it"
			if !strings.Contains(stderr, want) || strings.Contains(stderr, "usage error") || strings.Count(stderr, "\n") != 1 {
				t.Fatalf("stderr = %q, want one warning: %q", stderr, want)
			}
		})
	}
}

func TestAliasFileWithTrailingNewline(t *testing.T) {
	_, home := isolatedWorkspace(t)
	jdk := writeFakeManagedJDK(t, home)
	if err := os.WriteFile(filepath.Join(home, "work.alias"), []byte("temurin@21\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if stdout, stderr, code := executeRoot(t, "alias", "work"); code != exitSuccess || stdout != "temurin@21\n" {
		t.Fatalf("alias work: exit %d, stdout %q, stderr %q, want temurin@21 on one line", code, stdout, stderr)
	}
	if stdout, stderr, code := executeRoot(t, "which", "work"); code != exitSuccess || !strings.Contains(stdout, filepath.Base(jdk)) {
		t.Fatalf("which work: exit %d, stdout %q, stderr %q, want %s", code, stdout, stderr, jdk)
	}
}

func TestAliasFileWithEmbeddedNewline(t *testing.T) {
	_, home := isolatedWorkspace(t)
	writeFakeManagedJDK(t, home)
	_, missingStderr, missingCode := executeRoot(t, "which", "work")
	if err := os.WriteFile(filepath.Join(home, "work.alias"), []byte("temurin@21\necho compromised"), 0o600); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, code := executeRoot(t, "alias", "work")
	if code != exitUsage || stdout != "" || !strings.Contains(stderr, `alias "work" is invalid`) || strings.Contains(stderr, home) {
		t.Fatalf("alias work: exit %d, stdout %q, stderr %q, want exit %d naming the invalid alias without a path", code, stdout, stderr, exitUsage)
	}
	if _, stderr, code := executeRoot(t, "which", "work"); code != missingCode || stderr != missingStderr {
		t.Fatalf("which work: exit %d, stderr %q, want the missing-alias result: exit %d, stderr %q", code, stderr, missingCode, missingStderr)
	}
}
