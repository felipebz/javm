package command

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/felipebz/javm/discovery"
)

// isolateCurrentEnvironment clears the variables current reads, so the
// caller's shell cannot change the result, and stubs lookPath to find nothing.
func isolateCurrentEnvironment(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("JAVM_HOME", home)
	for _, key := range []string{"JAVM_ACTIVE_JAVA_HOME", "JAVA_HOME_BEFORE_JAVM", "JAVA_HOME", "PATH"} {
		t.Setenv(key, "")
		os.Unsetenv(key)
	}
	originalLookPath := lookPath
	lookPath = func(string) (string, error) { return "", exec.ErrNotFound }
	t.Cleanup(func() { lookPath = originalLookPath })
	cleanup := setupMockLs()
	t.Cleanup(cleanup)
	mockLsResult = nil
	mockLsError = nil
	return home
}

// makeFakeJDK creates root/bin/java (under Contents/Home when macOS is true).
func makeFakeJDK(t *testing.T, root string, macOS bool) string {
	t.Helper()
	home := root
	if macOS {
		home = filepath.Join(root, "Contents", "Home")
	}
	java := filepath.Join(home, "bin", "java")
	if runtime.GOOS == "windows" {
		java += ".exe"
	}
	if err := os.MkdirAll(filepath.Dir(java), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(java, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return home
}

func runCurrent(t *testing.T) (string, error) {
	t.Helper()
	var stdout bytes.Buffer
	cmd := NewCurrentCommand()
	cmd.SetOut(&stdout)
	// The root command silences usage for runtime errors.
	cmd.SilenceUsage = true
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs(nil)
	err := cmd.ExecuteContext(context.Background())
	return stdout.String(), err
}

func TestCurrentReportsActiveJDK(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, javmHome string)
		want  string
	}{
		{
			name: "managed JDK through the marker",
			setup: func(t *testing.T, javmHome string) {
				root := filepath.Join(javmHome, "jdk", "temurin@21.0.4")
				makeFakeJDK(t, root, false)
				mockLsResult = []discovery.JDK{{Identifier: "temurin@21.0.4", Source: "javm", Path: root}}
				t.Setenv("JAVM_ACTIVE_JAVA_HOME", root)
			},
			want: "temurin@21.0.4",
		},
		{
			name: "discovered JDK outside JAVM_HOME through the marker",
			setup: func(t *testing.T, _ string) {
				root := filepath.Join(t.TempDir(), "java-21-openjdk-amd64")
				makeFakeJDK(t, root, false)
				mockLsResult = []discovery.JDK{{Identifier: "ubuntu-system@21", Source: "system", Path: root}}
				t.Setenv("JAVM_ACTIVE_JAVA_HOME", root)
			},
			want: "ubuntu-system@21",
		},
		{
			name: "macOS Contents/Home layout through the marker",
			setup: func(t *testing.T, javmHome string) {
				root := filepath.Join(javmHome, "jdk", "zulu@21.0.4")
				home := makeFakeJDK(t, root, true)
				mockLsResult = []discovery.JDK{{Identifier: "zulu@21.0.4", Source: "javm", Path: root}}
				t.Setenv("JAVM_ACTIVE_JAVA_HOME", home)
			},
			want: "zulu@21.0.4",
		},
		{
			name: "macOS Contents/Home layout through PATH",
			setup: func(t *testing.T, javmHome string) {
				root := filepath.Join(javmHome, "jdk", "zulu@21.0.4")
				home := makeFakeJDK(t, root, true)
				mockLsResult = []discovery.JDK{{Identifier: "zulu@21.0.4", Source: "javm", Path: root}}
				lookPath = func(string) (string, error) { return filepath.Join(home, "bin", "java"), nil }
			},
			want: "zulu@21.0.4",
		},
		{
			// A javm link and the discovered JDK it points to share a home.
			name: "javm link preferred over discovered JDK with the same home",
			setup: func(t *testing.T, javmHome string) {
				if runtime.GOOS == "windows" {
					t.Skip("creating symlinks requires privileges on Windows")
				}
				target := filepath.Join(t.TempDir(), "java-21-openjdk-amd64")
				makeFakeJDK(t, target, false)
				link := filepath.Join(javmHome, "jdk", "system@21")
				if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, link); err != nil {
					t.Fatal(err)
				}
				mockLsResult = []discovery.JDK{
					{Identifier: "ubuntu-system@21", Source: "system", Path: target},
					{Identifier: "system@21", Source: "javm", Path: link},
				}
				t.Setenv("JAVM_ACTIVE_JAVA_HOME", target)
			},
			want: "system@21",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			javmHome := isolateCurrentEnvironment(t)
			tt.setup(t, javmHome)
			got, err := runCurrent(t)
			if err != nil {
				t.Fatalf("current error = %v", err)
			}
			if got != tt.want+"\n" {
				t.Fatalf("stdout = %q, want %q", got, tt.want+"\n")
			}
		})
	}
}

// On Linux /usr/bin/java usually reaches the JDK through /etc/alternatives.
func TestCurrentFollowsJavaSymlinkOnPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks requires privileges on Windows")
	}
	isolateCurrentEnvironment(t)
	lookPath = exec.LookPath
	root := filepath.Join(t.TempDir(), "java-17-openjdk-amd64")
	makeFakeJDK(t, root, false)
	mockLsResult = []discovery.JDK{{Identifier: "ubuntu-system@17", Source: "system", Path: root}}

	alternatives := t.TempDir()
	if err := os.Symlink(filepath.Join(root, "bin", "java"), filepath.Join(alternatives, "java")); err != nil {
		t.Fatal(err)
	}
	usrBin := t.TempDir()
	if err := os.Symlink(filepath.Join(alternatives, "java"), filepath.Join(usrBin, "java")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", usrBin)

	got, err := runCurrent(t)
	if err != nil {
		t.Fatalf("current error = %v", err)
	}
	if got != "ubuntu-system@17\n" {
		t.Fatalf("stdout = %q, want ubuntu-system@17", got)
	}
}

func TestCurrentFailsWhenNoKnownJDKIsActive(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T) string
		wantMessage func(path string) string
	}{
		{
			name: "marker names a home no JDK matches",
			setup: func(t *testing.T) string {
				gone := filepath.Join(t.TempDir(), "uninstalled-jdk")
				t.Setenv("JAVM_ACTIVE_JAVA_HOME", gone)
				return gone
			},
			wantMessage: func(path string) string { return path },
		},
		{
			name:        "no java on PATH",
			setup:       func(*testing.T) string { return "" },
			wantMessage: func(string) string { return "no JDK known to javm is active in this shell" },
		},
		{
			name: "java on PATH matches no JDK",
			setup: func(t *testing.T) string {
				unknown := makeFakeJDK(t, filepath.Join(t.TempDir(), "unknown-jdk"), false)
				java := filepath.Join(unknown, "bin", "java")
				lookPath = func(string) (string, error) { return java, nil }
				resolved, err := filepath.EvalSymlinks(java)
				if err != nil {
					t.Fatal(err)
				}
				return resolved
			},
			wantMessage: func(path string) string { return path },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			javmHome := isolateCurrentEnvironment(t)
			root := filepath.Join(javmHome, "jdk", "temurin@21.0.4")
			makeFakeJDK(t, root, false)
			mockLsResult = []discovery.JDK{{Identifier: "temurin@21.0.4", Source: "javm", Path: root}}
			path := tt.setup(t)

			got, err := runCurrent(t)
			if !errors.Is(err, ErrNotFound) {
				t.Fatalf("current error = %v, want ErrNotFound", err)
			}
			if got != "" {
				t.Fatalf("stdout = %q, want nothing", got)
			}
			if want := tt.wantMessage(path); !strings.Contains(err.Error(), want) {
				t.Fatalf("error %q does not mention %q", err, want)
			}
		})
	}
}
