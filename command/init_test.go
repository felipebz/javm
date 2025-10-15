package command

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// --- Fake the getExecutablePath and writePowerShellInitScript for testability ---

var testExecutablePath = "/test/javm"
var fakePowerShellTempFile = "/tmp/fake_javm.ps1"

func init() {
	getExecutablePath = func() (string, error) { return testExecutablePath, nil }
	writePowerShellInitScript = func(script string) (string, error) { return fakePowerShellTempFile, nil }
}

func TestInitCommand_Bash(t *testing.T) {
	// Isolate from any real user config by pointing JAVM_HOME to a temp dir
	tmp := t.TempDir()
	oldHome, had := os.LookupEnv("JAVM_HOME")
	os.Setenv("JAVM_HOME", tmp)
	if had {
		t.Cleanup(func() { os.Setenv("JAVM_HOME", oldHome) })
	} else {
		t.Cleanup(func() { os.Unsetenv("JAVM_HOME") })
	}

	cmd := NewInitCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"bash"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, testExecutablePath) {
		t.Errorf("script does not contain the executable path, got: %s", output)
	}
	if strings.Contains(output, "::JAVM::") {
		t.Errorf("placeholder was not replaced: %s", output)
	}
}

func TestInitCommand_PowerShell(t *testing.T) {
	// Isolate from any real user config by pointing JAVM_HOME to a temp dir
	tmp := t.TempDir()
	oldHome, had := os.LookupEnv("JAVM_HOME")
	os.Setenv("JAVM_HOME", tmp)
	if had {
		t.Cleanup(func() { os.Setenv("JAVM_HOME", oldHome) })
	} else {
		t.Cleanup(func() { os.Unsetenv("JAVM_HOME") })
	}

	cmd := NewInitCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"pwsh"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buf.String()
	want := "& '" + fakePowerShellTempFile + "'\n"
	if output != want {
		t.Errorf("unexpected pwsh output, got: %q, want: %q", output, want)
	}
}

func TestInitCommand_UnsupportedShell(t *testing.T) {
	cmd := NewInitCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"foo"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "unsupported shell") {
		t.Errorf("expected error for unsupported shell, got: %v", err)
	}
}

func TestSortedShells(t *testing.T) {
	keys := sortedShells()
	want := []string{"bash", "fish", "powershell", "pwsh", "zsh"}
	for _, k := range want {
		found := false
		for _, v := range keys {
			if v == k {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("sortedShells() missing: %s", k)
		}
	}
}

func TestRealWritePowerShellInitScript(t *testing.T) {
	content := "# fake script for test\n"
	path, err := realWritePowerShellInitScript(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer os.Remove(path)

	// Check that the file exists and contains the expected content
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("could not read file: %v", err)
	}
	if string(data) != content {
		t.Errorf("file content mismatch: got %q, want %q", string(data), content)
	}
}
