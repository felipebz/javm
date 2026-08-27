package main

import (
	"bytes"
	"fmt"
	"os"
	osExec "os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/felipebz/javm/discoapi"
	log "github.com/sirupsen/logrus"
)

func TestRootExecPassesChildArgumentsOpaque(t *testing.T) {
	probe := createRootExecFixture(t)

	tests := []struct {
		name      string
		prefix    []string
		selector  string
		arguments []string
	}{
		{name: "implicit Maven profile", arguments: []string{"-Pcustom-profile", "package"}},
		{name: "implicit system property", arguments: []string{"-Dfoo=bar"}},
		{name: "implicit stacktrace", arguments: []string{"--stacktrace"}},
		{name: "implicit JVM option", arguments: []string{"-Xmx2g"}},
		{name: "explicit selector", selector: "21", arguments: []string{"-Pcustom-profile"}},
		{name: "root persistent flag before command", prefix: []string{"--quiet"}, arguments: []string{"-Pcustom-profile"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			logger := log.New()
			root := newRootCommand(application{
				out:    &stdout,
				err:    &stderr,
				logger: logger,
				client: discoapi.NewClient(),
			})

			args := append([]string{}, tt.prefix...)
			args = append(args, "exec")
			if tt.selector != "" {
				args = append(args, "--jdk", tt.selector)
			}
			args = append(args, filepath.Base(probe))
			args = append(args, tt.arguments...)
			root.SetArgs(args)

			if err := root.Execute(); err != nil {
				t.Fatalf("root.Execute() = %v\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String())
			}

			want := fmt.Sprintf("ARGS=%q\n", tt.arguments)
			if !strings.Contains(stdout.String(), want) {
				t.Fatalf("child arguments = %q, want line %q", stdout.String(), want)
			}
		})
	}
}

func TestPowerShellIntegrationUsesCanonicalExecSyntax(t *testing.T) {
	pwsh, err := osExec.LookPath("pwsh")
	if err != nil {
		t.Skip("pwsh is not installed")
	}
	goBinary, err := osExec.LookPath("go")
	if err != nil {
		t.Skip("go is not installed")
	}
	probe := createRootExecFixture(t)

	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	binaryName := "javm"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	binary := filepath.Join(t.TempDir(), binaryName)
	build := osExec.Command(goBinary, "build", "-o", binary, ".")
	build.Dir = filepath.Dir(testFile)
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build javm: %v\n%s", err, output)
	}

	runner := filepath.Join(t.TempDir(), "pwsh-exec-test.ps1")
	script := fmt.Sprintf(`
$ErrorActionPreference = 'Stop'
$init = & $env:JAVM_BINARY init pwsh
Invoke-Expression ($init -join [Environment]::NewLine)

javm exec %s -Pcustom-profile package
javm exec --jdk 21 %s -Dfoo=bar test

javm use 21
if (-not $env:JAVA_HOME) { throw 'javm use did not set JAVA_HOME' }
javm deactivate
if ($env:JAVA_HOME -ne '/parent/java') { throw "javm deactivate restored $env:JAVA_HOME instead of /parent/java" }
`, filepath.Base(probe), filepath.Base(probe))
	if err := os.WriteFile(runner, []byte(script), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd := osExec.Command(pwsh, "-NoProfile", "-NonInteractive", "-File", runner)
	cmd.Env = append(os.Environ(), "JAVM_BINARY="+binary)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("PowerShell integration failed: %v\n%s", err, output)
	}
	text := string(output)
	if !strings.Contains(text, `ARGS=["-Pcustom-profile" "package"]`) {
		t.Fatalf("implicit exec arguments were not preserved:\n%s", text)
	}
	if !strings.Contains(text, `ARGS=["-Dfoo=bar" "test"]`) {
		t.Fatalf("explicit --jdk exec arguments were not preserved:\n%s", text)
	}
}

func createRootExecFixture(t *testing.T) string {
	t.Helper()

	home := t.TempDir()
	root := filepath.Join(home, "jdk", "temurin@21.0.1")
	jdkHome := root
	if runtime.GOOS == "darwin" {
		jdkHome = filepath.Join(root, "Contents", "Home")
	}
	bin := filepath.Join(jdkHome, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}

	java := "java"
	probe := "probe"
	if runtime.GOOS == "windows" {
		java += ".exe"
		probe += ".exe"
	}
	if err := os.WriteFile(filepath.Join(bin, java), []byte("fake java"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(jdkHome, "release"), []byte("JAVA_VERSION=\"21.0.1\"\nJAVA_VENDOR=\"TestVendor\"\nOS_ARCH=\"x64\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	build := osExec.Command("go", "build", "-o", filepath.Join(bin, probe), "./command/testdata/exec-helper")
	build.Dir = filepath.Dir(testFile)
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build exec helper: %v\n%s", err, output)
	}

	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, ".java-version"), []byte("21\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(workspace); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	t.Setenv("JAVM_HOME", home)
	t.Setenv("PATH", t.TempDir())
	t.Setenv("JAVA_HOME", "/parent/java")
	t.Setenv("JAVM_TEST_EXIT", "")

	return filepath.Join(bin, probe)
}
