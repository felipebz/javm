package command

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
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

func TestInitUsesDefaultAsData(t *testing.T) {
	t.Setenv("JAVM_HOME", t.TempDir())
	selector := "temurin@21"
	if err := SetDefaultVersion(selector); err != nil {
		t.Fatal(err)
	}

	for _, shell := range []string{"bash", "zsh", "fish", "nu", "cmd"} {
		t.Run(shell, func(t *testing.T) {
			cmd := NewInitCommand()
			var output bytes.Buffer
			cmd.SetOut(&output)
			cmd.SetArgs([]string{shell})
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			defaultInvocation := "javm use --default"
			if shell == "cmd" {
				defaultInvocation = "use --default"
			}
			if !strings.Contains(output.String(), defaultInvocation) {
				t.Fatalf("%s init does not invoke the static default path", shell)
			}
			if strings.Contains(output.String(), selector) {
				t.Fatalf("%s init interpolated the persisted selector", shell)
			}
		})
	}
}

func TestInitCMDInitializesDefaultWithoutRecursion(t *testing.T) {
	t.Setenv("JAVM_HOME", t.TempDir())
	if err := SetDefaultVersion("temurin@21"); err != nil {
		t.Fatal(err)
	}

	cmd := NewInitCommand()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"cmd"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	script := output.String()
	if !strings.Contains(script, `"%_JAVM_EXECUTABLE%" --fd3 "%_JAVM_ENV_FILE%" use --default`) {
		t.Fatal("CMD init does not invoke the executable directly for default initialization")
	}
	if strings.Contains(script, "call javm use --default") {
		t.Fatal("CMD init recursively invokes its own wrapper")
	}
	if !strings.Contains(script, "if not defined _JAVM_DEFAULT_INITIALIZED goto javm_initialize_default") {
		t.Fatal("CMD init does not guard default initialization per session")
	}
}

func TestInitCMDWithoutDefaultKeepsRuntimeInitialization(t *testing.T) {
	t.Setenv("JAVM_HOME", t.TempDir())
	cmd := NewInitCommand()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"cmd"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), "::JAVM_DEFAULT_INIT::") {
		t.Fatal("CMD init left the default initialization placeholder in the wrapper")
	}
	if !strings.Contains(output.String(), "use --default") {
		t.Fatal("CMD init cannot pick up a default configured after wrapper generation")
	}
}

func TestInitPowerShellUsesDefaultAsData(t *testing.T) {
	t.Setenv("JAVM_HOME", t.TempDir())
	selector := "temurin@21"
	if err := SetDefaultVersion(selector); err != nil {
		t.Fatal(err)
	}

	previousWriter := writePowerShellInitScript
	var generatedScript string
	writePowerShellInitScript = func(script string) (string, error) {
		generatedScript = script
		return fakePowerShellTempFile, nil
	}
	t.Cleanup(func() { writePowerShellInitScript = previousWriter })

	cmd := NewInitCommand()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetArgs([]string{"pwsh"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(generatedScript, "javm use --default") {
		t.Fatal("PowerShell init does not invoke the static default path")
	}
	if strings.Contains(generatedScript, selector) {
		t.Fatal("PowerShell init interpolated the persisted selector")
	}
}

func TestInitRejectsTamperedDefault(t *testing.T) {
	home := t.TempDir()
	t.Setenv("JAVM_HOME", home)
	payload := "17\necho compromised"
	if err := os.WriteFile(filepath.Join(home, "default-version"), []byte(payload), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd := NewInitCommand()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"bash"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("init accepted a tampered default-version file")
	}
	if strings.Contains(output.String(), payload) {
		t.Fatal("init emitted the tampered selector")
	}
}

func TestEscapeExecutablePath(t *testing.T) {
	tests := []struct {
		shell string
		path  string
		want  string
	}{
		{shell: "bash", path: `/tmp/$(command)\"`, want: `/tmp/\$(command)\\\"`},
		{shell: "fish", path: `/tmp/$(command)\"`, want: `/tmp/\$\(command\)\\\"`},
		{shell: "nu", path: `/tmp/$path\"`, want: `/tmp/$path\\\"`},
		{shell: "pwsh", path: `C:\it's\javm.exe`, want: `C:\it''s\javm.exe`},
		{shell: "cmd", path: `C:\100%\javm.exe`, want: `C:\100%%\javm.exe`},
	}
	for _, tt := range tests {
		t.Run(tt.shell, func(t *testing.T) {
			got, err := escapeExecutablePath(tt.shell, tt.path)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("escaped path = %q, want %q", got, tt.want)
			}
		})
	}

	for _, path := range []string{"bad\npath", "bad\rpath", "bad\x00path"} {
		if _, err := escapeExecutablePath("bash", path); err == nil {
			t.Fatalf("escapeExecutablePath(%q) did not reject control characters", path)
		}
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

func TestInitCommand_CMD(t *testing.T) {
	t.Setenv("JAVM_HOME", t.TempDir())
	oldExecutablePath := testExecutablePath
	testExecutablePath = `C:\Program Files\javm & tools\100%^\javm.exe`
	t.Cleanup(func() { testExecutablePath = oldExecutablePath })

	cmd := NewInitCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"cmd"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `set "_JAVM_EXECUTABLE=C:\Program Files\javm & tools\100%%^\javm.exe"`) {
		t.Errorf("cmd wrapper does not safely embed the executable path, got: %s", output)
	}
	if !strings.Contains(output, `setlocal DisableDelayedExpansion`) {
		t.Errorf("cmd wrapper must preserve exclamation marks in paths, got: %s", output)
	}
	if !strings.Contains(output, `exit /b %_JAVM_EXIT_CODE%`) {
		t.Errorf("cmd wrapper does not propagate the exit code, got: %s", output)
	}
	if strings.Contains(output, "::JAVM::") {
		t.Errorf("placeholder was not replaced: %s", output)
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
	want := []string{"bash", "cmd", "fish", "nu", "powershell", "pwsh", "zsh"}
	for _, k := range want {
		found := slices.Contains(keys, k)
		if !found {
			t.Errorf("sortedShells() missing: %s", k)
		}
	}
}

func TestInitCommand_Nushell(t *testing.T) {
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
	cmd.SetArgs([]string{"nu"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, testExecutablePath) {
		t.Errorf("script does not contain the executable path, got: %s", output)
	}
	if !strings.Contains(output, "def --env --wrapped javm") {
		t.Errorf("script does not look like a nushell script, got: %s", output)
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
