package main

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/felipebz/javm/discoapi"
	log "github.com/sirupsen/logrus"
)

// isolatedWorkspace runs the test from an empty directory with its own
// JAVM_HOME and no activation state inherited from the caller.
func isolatedWorkspace(t *testing.T) (workspace, home string) {
	t.Helper()
	workspace = t.TempDir()
	t.Chdir(workspace)
	home = t.TempDir()
	t.Setenv("JAVM_HOME", home)
	for _, key := range []string{"JAVM_ACTIVE_JAVA_HOME", "JAVA_HOME_BEFORE_JAVM", "JAVA_HOME"} {
		t.Setenv(key, "")
		os.Unsetenv(key)
	}
	return workspace, home
}

func executeRoot(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	// The same formatter as main, so tests see the messages users see.
	logger := log.New()
	logger.SetFormatter(&simpleFormatter{})
	root := newRootCommand(application{
		out:    &stdout,
		err:    &stderr,
		logger: logger,
		client: discoapi.NewClient(),
	})
	root.SetArgs(args)
	err := root.Execute()
	return stdout.String(), stderr.String(), exitCode(err)
}

var selectorCommands = [][]string{
	{"install"},
	{"use"},
	{"which"},
	{"exec", "java", "-version"},
}

func TestCommandsWithoutSelectorReportMissingJavaVersion(t *testing.T) {
	for _, args := range selectorCommands {
		t.Run(args[0], func(t *testing.T) {
			workspace, _ := isolatedWorkspace(t)
			skipIfAncestorHasJavaVersion(t, workspace)
			stdout, stderr, code := executeRoot(t, args...)
			if code != exitUsage {
				t.Fatalf("exit code = %d, want %d\nstdout: %s\nstderr: %s", code, exitUsage, stdout, stderr)
			}
			if strings.Contains(stdout, "Usage:") {
				t.Fatalf("printed help instead of an error:\n%s", stdout)
			}
			physical, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(stderr, ".java-version") || (!strings.Contains(stderr, workspace) && !strings.Contains(stderr, physical)) {
				t.Fatalf("stderr = %q, want a message naming .java-version and %s", stderr, workspace)
			}
		})
	}
}

func TestCommandsWithoutSelectorReportEmptyJavaVersion(t *testing.T) {
	for _, content := range []string{"", " \n\t\n"} {
		for _, args := range selectorCommands {
			t.Run(args[0]+"/"+strings.ReplaceAll(content, "\n", `\n`), func(t *testing.T) {
				isolatedWorkspace(t)
				if err := os.WriteFile(".java-version", []byte(content), 0o600); err != nil {
					t.Fatal(err)
				}
				_, stderr, code := executeRoot(t, args...)
				if code != exitUsage {
					t.Fatalf("exit code = %d, want %d\nstderr: %s", code, exitUsage, stderr)
				}
				if !strings.Contains(stderr, ".java-version is empty") {
					t.Fatalf("stderr = %q, want it to say .java-version is empty", stderr)
				}
			})
		}
	}
}

func TestCommandsWithoutSelectorReportUnreadableJavaVersion(t *testing.T) {
	for _, args := range selectorCommands {
		t.Run(args[0], func(t *testing.T) {
			isolatedWorkspace(t)
			if err := os.Mkdir(".java-version", 0o755); err != nil {
				t.Fatal(err)
			}
			_, stderr, code := executeRoot(t, args...)
			if code != exitUsage {
				t.Fatalf("exit code = %d, want %d\nstderr: %s", code, exitUsage, stderr)
			}
			if !strings.Contains(stderr, "read ") || !strings.Contains(stderr, ".java-version") || strings.Contains(stderr, "is empty") {
				t.Fatalf("stderr = %q, want a read error naming .java-version", stderr)
			}
		})
	}
}

func TestExecReportsMissingCommandBeforeSelector(t *testing.T) {
	isolatedWorkspace(t)
	_, stderr, code := executeRoot(t, "exec")
	if code != exitUsage || !strings.Contains(stderr, "no command specified") {
		t.Fatalf("exit code = %d, stderr = %q, want the no command specified error", code, stderr)
	}
}

func TestSelectorCommandsHelpStillExitsZero(t *testing.T) {
	for _, name := range []string{"install", "use", "which", "exec"} {
		t.Run(name, func(t *testing.T) {
			isolatedWorkspace(t)
			stdout, stderr, code := executeRoot(t, name, "--help")
			if code != exitSuccess || !strings.Contains(stdout+stderr, "Usage:") {
				t.Fatalf("exit code = %d, output = %q, want help with exit 0", code, stdout+stderr)
			}
		})
	}
}

func TestUseDefaultWithoutDefaultStaysSilent(t *testing.T) {
	isolatedWorkspace(t)
	envFile := filepath.Join(t.TempDir(), "environment")
	stdout, stderr, code := executeRoot(t, "use", "--fd3", envFile, "--default")
	if code != exitSuccess || stdout != "" || stderr != "" {
		t.Fatalf("exit code = %d, stdout = %q, stderr = %q, want a silent success", code, stdout, stderr)
	}
	if data, err := os.ReadFile(envFile); err == nil && len(data) != 0 {
		t.Fatalf("use --default wrote records without a default: %q", data)
	}
}

// writeFakeManagedJDK creates a managed temurin@21.0.1 that discovery accepts
// without running java, and returns its path.
func writeFakeManagedJDK(t *testing.T, home string) string {
	t.Helper()
	jdk := filepath.Join(home, "jdk", "temurin@21.0.1")
	jdkHome := jdk
	if runtime.GOOS == "darwin" {
		jdkHome = filepath.Join(jdk, "Contents", "Home")
	}
	java := filepath.Join(jdkHome, "bin", "java")
	if runtime.GOOS == "windows" {
		java += ".exe"
	}
	if err := os.MkdirAll(filepath.Dir(java), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(java, []byte("fake java"), 0o755); err != nil {
		t.Fatal(err)
	}
	release := "JAVA_VERSION=\"21.0.1\"\nJAVA_VENDOR=\"TestVendor\"\nOS_ARCH=\"x64\"\n"
	if err := os.WriteFile(filepath.Join(jdkHome, "release"), []byte(release), 0o644); err != nil {
		t.Fatal(err)
	}
	return jdk
}

func TestWhichUsesValidJavaVersion(t *testing.T) {
	_, home := isolatedWorkspace(t)
	jdk := writeFakeManagedJDK(t, home)
	if err := os.WriteFile(".java-version", []byte("  temurin@21\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, code := executeRoot(t, "which")
	if code != exitSuccess || stdout != jdk+"\n" {
		t.Fatalf("exit code = %d, stdout = %q, stderr = %q, want %s", code, stdout, stderr, jdk)
	}
}

// Commands run from a subdirectory of a project use the project's
// .java-version.
func TestSelectorCommandsUseParentJavaVersion(t *testing.T) {
	workspace, home := isolatedWorkspace(t)
	jdk := writeFakeManagedJDK(t, home)
	if err := os.WriteFile(".java-version", []byte("temurin@21\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	module := filepath.Join(workspace, "module-a", "src")
	if err := os.MkdirAll(module, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(module)

	stdout, stderr, code := executeRoot(t, "which")
	if code != exitSuccess || stdout != jdk+"\n" {
		t.Fatalf("which: exit code = %d, stdout = %q, stderr = %q, want %s", code, stdout, stderr, jdk)
	}

	// Without a real JVM, a missing command shows that exec resolved the
	// selector from the parent and selected the JDK before looking it up.
	_, stderr, code = executeRoot(t, "exec", "javm-test-missing-command")
	if code != exitNotFound || !strings.Contains(stderr, `"javm-test-missing-command" was not found in the environment for temurin@21`) {
		t.Fatalf("exec: exit code = %d, stderr = %q, want the command lookup to fail for temurin@21", code, stderr)
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
