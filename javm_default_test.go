package main

import (
	"bytes"
	"os"
	osExec "os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDefaultRejectsSelectorWithoutInstalledJDK(t *testing.T) {
	_, home := isolatedWorkspace(t)
	defaultFile := filepath.Join(home, "default-version")

	_, stderr, code := executeRoot(t, "default", "99")
	if code != exitNotFound {
		t.Fatalf("exit code = %d, want %d\nstderr: %s", code, exitNotFound, stderr)
	}
	if !strings.Contains(stderr, `"99"`) || !strings.Contains(stderr, "javm install 99") {
		t.Fatalf("stderr = %q, want the selector and an install hint", stderr)
	}
	if _, err := os.Stat(defaultFile); !os.IsNotExist(err) {
		t.Fatalf("default-version was written: %v", err)
	}

	if err := os.WriteFile(defaultFile, []byte("17"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, code := executeRoot(t, "default", "99"); code != exitNotFound {
		t.Fatalf("exit code = %d, want %d", code, exitNotFound)
	}
	if data, err := os.ReadFile(defaultFile); err != nil || string(data) != "17" {
		t.Fatalf("default-version = %q, %v, want it unchanged", data, err)
	}
}

func TestDefaultStoresSelectorOfInstalledJDK(t *testing.T) {
	_, home := isolatedWorkspace(t)
	writeFakeManagedJDK(t, home)

	stdout, stderr, code := executeRoot(t, "default", "21")
	if code != exitSuccess {
		t.Fatalf("exit code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
	}
	// The selector is stored, not the resolved JDK, so the default follows
	// updates of the 21 line.
	if data, err := os.ReadFile(filepath.Join(home, "default-version")); err != nil || string(data) != "21" {
		t.Fatalf("default-version = %q, %v, want 21", data, err)
	}
}

func TestUseDefaultWarnsWhenDefaultIsUnusable(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    func(home string) string
	}{
		{name: "not installed", content: "99", want: func(string) string { return `default JDK "99" is not installed` }},
		{name: "invalid file", content: "not a version", want: func(home string) string { return filepath.Join(home, "default-version") }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, home := isolatedWorkspace(t)
			if err := os.WriteFile(filepath.Join(home, "default-version"), []byte(tt.content), 0o600); err != nil {
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
			if want := tt.want(home); strings.Count(stderr, want) != 1 || strings.Count(stderr, "\n") != 1 {
				t.Fatalf("stderr = %q, want one warning naming %s", stderr, want)
			}
		})
	}
}

func TestUseDefaultActivatesInstalledDefault(t *testing.T) {
	_, home := isolatedWorkspace(t)
	jdk := writeFakeManagedJDK(t, home)
	if err := os.WriteFile(filepath.Join(home, "default-version"), []byte("temurin@21"), 0o600); err != nil {
		t.Fatal(err)
	}
	envFile := filepath.Join(t.TempDir(), "environment")

	_, stderr, code := executeRoot(t, "use", "--fd3", envFile, "--default")
	data, err := os.ReadFile(envFile)
	if code != exitSuccess || err != nil || !strings.Contains(string(data), "SET\tJAVA_HOME\t") || !strings.Contains(string(data), filepath.Base(jdk)) {
		t.Fatalf("exit code = %d, records = %q, %v, stderr = %q, want the default activated", code, data, err, stderr)
	}
}

// A default that points to a JDK that is not installed must not break the
// eval line in the user's shell startup file.
func TestShellStartupSurvivesUninstalledDefault(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell wrappers")
	}
	posix := func(shell string) string {
		return `ORIG_PATH=$PATH
export JAVA_HOME=/orig
eval "$("$JAVM_BINARY" init ` + shell + `)"
rc=$?
[ "$rc" -eq 0 ] || { echo "init exited with $rc"; exit 1; }
[ "$JAVA_HOME" = /orig ] || { echo "JAVA_HOME=[$JAVA_HOME], want [/orig]"; exit 1; }
[ "$PATH" = "$ORIG_PATH" ] || { echo "PATH changed to [$PATH]"; exit 1; }
`
	}
	scripts := map[string]struct {
		args   []string
		script string
	}{
		"bash": {args: []string{"--noprofile", "--norc"}, script: posix("bash")},
		"zsh":  {args: []string{"-f"}, script: posix("zsh")},
		"fish": {args: []string{"--no-config"}, script: `set -l orig_path (string join : $PATH)
set -gx JAVA_HOME /orig
$JAVM_BINARY init fish | source
set -l rc $status
if test $rc -ne 0
    echo "init exited with $rc"; exit 1
end
if test "$JAVA_HOME" != /orig
    echo "JAVA_HOME=[$JAVA_HOME], want [/orig]"; exit 1
end
if test (string join : $PATH) != "$orig_path"
    echo "PATH changed to ["(string join : $PATH)"]"; exit 1
end
`},
	}
	shells := map[string]string{}
	for name := range scripts {
		if path, err := osExec.LookPath(name); err == nil {
			shells[name] = path
		}
	}
	if len(shells) == 0 {
		t.Skip("bash, zsh and fish are not installed")
	}
	goBinary, err := osExec.LookPath("go")
	if err != nil {
		t.Skip("go is not installed")
	}
	shellPath := os.Getenv("PATH")
	_, home := isolatedWorkspace(t)
	if err := os.WriteFile(filepath.Join(home, "default-version"), []byte("99"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	binary := filepath.Join(t.TempDir(), "javm")
	build := osExec.Command(goBinary, "build", "-o", binary, ".")
	build.Dir = filepath.Dir(testFile)
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build javm: %v\n%s", err, output)
	}

	for name, shell := range shells {
		t.Run(name, func(t *testing.T) {
			runner := filepath.Join(t.TempDir(), "startup-test")
			if err := os.WriteFile(runner, []byte(scripts[name].script), 0o600); err != nil {
				t.Fatal(err)
			}
			cmd := osExec.Command(shell, append(scripts[name].args, runner)...)
			cmd.Env = append(os.Environ(), "PATH="+shellPath, "JAVM_BINARY="+binary)
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			if err := cmd.Run(); err != nil {
				t.Fatalf("%s startup failed: %v\nstdout: %s\nstderr: %s", name, err, stdout.String(), stderr.String())
			}
			if !strings.Contains(stderr.String(), `default JDK "99" is not installed`) {
				t.Fatalf("stderr = %q, want the default warning", stderr.String())
			}
		})
	}
}
