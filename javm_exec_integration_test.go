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

func TestRootUseAcceptsLocalTransportFlag(t *testing.T) {
	createRootExecFixture(t)
	envFile := filepath.Join(t.TempDir(), "javm-env")
	var stdout, stderr bytes.Buffer
	root := newRootCommand(application{
		out:    &stdout,
		err:    &stderr,
		logger: log.New(),
		client: discoapi.NewClient(),
	})
	root.SetArgs([]string{"use", "--fd3", envFile, "21"})
	if err := root.Execute(); err != nil {
		t.Fatalf("root.Execute() = %v\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String())
	}
	data, err := os.ReadFile(envFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "SET\tJAVA_HOME\t") {
		t.Fatalf("use did not write environment updates through local --fd3: %q", data)
	}
}

func TestRootRejectsTransportFlagOutsideEnvironmentCommands(t *testing.T) {
	var stdout, stderr bytes.Buffer
	root := newRootCommand(application{
		out:    &stdout,
		err:    &stderr,
		logger: log.New(),
		client: discoapi.NewClient(),
	})
	root.SetArgs([]string{"--fd3", filepath.Join(t.TempDir(), "environment"), "install", "21"})
	if err := root.Execute(); err == nil || !strings.Contains(err.Error(), "unknown flag: --fd3") {
		t.Fatalf("root.Execute() = %v, want unknown transport flag outside use/deactivate", err)
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
	probe, fixtureHome := createRootExecFixtureWithHome(t)
	if err := os.WriteFile(filepath.Join(fixtureHome, "default-version"), []byte("21"), 0o600); err != nil {
		t.Fatal(err)
	}

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
if ($LASTEXITCODE -ne 0) { throw "PowerShell initialization failed with exit code $LASTEXITCODE" }

javm exec %s -Pcustom-profile package
javm exec --jdk 21 %s -Dfoo=bar test

javm use 21
if ($LASTEXITCODE -ne 0) { throw "javm use failed with exit code $LASTEXITCODE" }
if (-not $env:JAVA_HOME) { throw 'javm use did not set JAVA_HOME' }
javm deactivate
if ($LASTEXITCODE -ne 0) { throw "javm deactivate failed with exit code $LASTEXITCODE" }
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
	probe, _ := createRootExecFixtureWithHome(t)
	return probe
}

func createRootExecFixtureWithHome(t *testing.T) (string, string) {
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
	// A shell where javm already ran use would make the first use in these
	// tests look like a re-activation.
	for _, key := range []string{"JAVM_ACTIVE_JAVA_HOME", "JAVA_HOME_BEFORE_JAVM"} {
		t.Setenv(key, "")
		os.Unsetenv(key)
	}

	return filepath.Join(bin, probe), home
}

// A marker left behind for an uninstalled JDK makes current fail with exit
// code 3, and the error is printed once.
func TestRootCurrentReportsStaleMarker(t *testing.T) {
	createRootExecFixture(t)
	gone := filepath.Join(t.TempDir(), "uninstalled-jdk")
	t.Setenv("JAVM_ACTIVE_JAVA_HOME", gone)

	var stdout, stderr bytes.Buffer
	root := newRootCommand(application{
		out:    &stdout,
		err:    &stderr,
		logger: log.New(),
		client: discoapi.NewClient(),
	})
	root.SetArgs([]string{"current"})
	err := root.Execute()
	if code := exitCode(err); code != 3 {
		t.Fatalf("exitCode(%v) = %d, want 3", err, code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want nothing", stdout.String())
	}
	if strings.Count(stderr.String(), gone) != 1 || strings.Count(stderr.String(), "\n") != 1 {
		t.Fatalf("stderr = %q, want one line naming %s", stderr.String(), gone)
	}
}

// When use selected a JDK outside JAVM_HOME in the parent shell, exec must
// drop that JDK's bin instead of leaving it behind the selected one.
func TestRootExecRemovesPreviouslyActiveJDKFromPath(t *testing.T) {
	probe := createRootExecFixture(t)
	discovered := t.TempDir()
	discoveredBin := filepath.Join(discovered, "bin")
	userPath := os.Getenv("PATH")
	t.Setenv("PATH", discoveredBin+string(os.PathListSeparator)+userPath)
	t.Setenv("JAVA_HOME", discovered)
	t.Setenv("JAVM_ACTIVE_JAVA_HOME", discovered)

	var stdout, stderr bytes.Buffer
	root := newRootCommand(application{
		out:    &stdout,
		err:    &stderr,
		logger: log.New(),
		client: discoapi.NewClient(),
	})
	root.SetArgs([]string{"exec", filepath.Base(probe)})
	if err := root.Execute(); err != nil {
		t.Fatalf("root.Execute() = %v\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String())
	}
	want := "PATH=" + filepath.Dir(probe) + string(os.PathListSeparator) + userPath + "\n"
	if !strings.Contains(stdout.String(), want) {
		t.Fatalf("child output = %q, want line %q", stdout.String(), want)
	}
}

// The bash/zsh and fish wrappers read the --fd3 records with `read`, which
// skips a final line without a newline. That used to drop JAVA_HOME_BEFORE_JAVM,
// so deactivate kept the selected JDK instead of restoring JAVA_HOME.
func TestShellIntegrationDeactivateRestoresJavaHome(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell wrappers")
	}
	scripts := map[string]struct {
		args   []string
		script string
	}{
		"bash": {args: []string{"--noprofile", "--norc"}, script: posixDeactivateScript("bash")},
		"zsh":  {args: []string{"-f"}, script: posixDeactivateScript("zsh")},
		"fish": {args: []string{"--no-config"}, script: fishDeactivateScript},
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
	createRootExecFixture(t)
	gradleHome := createGradleJDK(t)

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
			runner := filepath.Join(t.TempDir(), "deactivate-test")
			if err := os.WriteFile(runner, []byte(scripts[name].script), 0o600); err != nil {
				t.Fatal(err)
			}
			cmd := osExec.Command(shell, append(scripts[name].args, runner)...)
			cmd.Env = append(os.Environ(), "PATH="+shellPath, "JAVM_BINARY="+binary, "GRADLE_USER_HOME="+gradleHome)
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("%s use/deactivate round trip failed: %v\n%s", name, err, output)
			}
		})
	}
}

// createGradleJDK registers a fake JDK outside JAVM_HOME that discovery finds
// as testvendor-gradle@17.
func createGradleJDK(t *testing.T) string {
	t.Helper()
	gradleHome := t.TempDir()
	jdk := filepath.Join(gradleHome, "jdks", "test-jdk-17")
	// Discovery expects the bundle layout on macOS.
	if runtime.GOOS == "darwin" {
		jdk = filepath.Join(jdk, "Contents", "Home")
	}
	if err := os.MkdirAll(filepath.Join(jdk, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(jdk, "bin", "java"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(jdk, "release"), []byte("JAVA_VERSION=\"17.0.9\"\nJAVA_VENDOR=\"TestVendor\"\nOS_ARCH=\"x64\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return gradleHome
}

func posixDeactivateScript(shell string) string {
	return `eval "$("$JAVM_BINARY" init ` + shell + `)"
ORIG_PATH=$PATH
export JAVA_HOME=/orig
unset JAVA_HOME_BEFORE_JAVM
unset JAVM_ACTIVE_JAVA_HOME
javm use 21 || exit 1
[ "$JAVA_HOME" != /orig ] || { echo "use did not change JAVA_HOME"; exit 1; }
[ "$JAVA_HOME_BEFORE_JAVM" = /orig ] || { echo "JAVA_HOME_BEFORE_JAVM=[$JAVA_HOME_BEFORE_JAVM], want [/orig]"; exit 1; }
javm deactivate || exit 1
[ "$JAVA_HOME" = /orig ] || { echo "JAVA_HOME=[$JAVA_HOME] after deactivate, want [/orig]"; exit 1; }
[ -z "${JAVA_HOME_BEFORE_JAVM+set}" ] || { echo "JAVA_HOME_BEFORE_JAVM is still set"; exit 1; }

# A discovered JDK lives outside JAVM_HOME; its bin must not stay in PATH.
javm use testvendor-gradle@17 || exit 1
javm use testvendor-gradle@17 || exit 1
[ "$(javm current)" = testvendor-gradle@17 ] || { echo "current=[$(javm current)], want [testvendor-gradle@17]"; exit 1; }
javm deactivate || exit 1
# The real PATH may have a system java that discovery knows; hide it.
PATH=/nonexistent "$JAVM_BINARY" current >/dev/null 2>&1
rc=$?
[ "$rc" -eq 3 ] || { echo "current exited with $rc after deactivate, want 3"; exit 1; }
[ "$PATH" = "$ORIG_PATH" ] || { echo "PATH=[$PATH] after deactivate, want [$ORIG_PATH]"; exit 1; }
[ "$JAVA_HOME" = /orig ] || { echo "JAVA_HOME=[$JAVA_HOME] after deactivate, want [/orig]"; exit 1; }
[ -z "${JAVM_ACTIVE_JAVA_HOME+set}" ] || { echo "JAVM_ACTIVE_JAVA_HOME is still set"; exit 1; }

# Without an original JAVA_HOME, deactivate unsets it instead of leaving it empty.
unset JAVA_HOME
javm use 21 || exit 1
javm use testvendor-gradle@17 || exit 1
javm deactivate || exit 1
[ -z "${JAVA_HOME+set}" ] || { echo "JAVA_HOME=[$JAVA_HOME] after deactivate, want unset"; exit 1; }
[ "$PATH" = "$ORIG_PATH" ] || { echo "PATH=[$PATH] after deactivate, want [$ORIG_PATH]"; exit 1; }
`
}

const fishDeactivateScript = `$JAVM_BINARY init fish | source
# The wrapper assigns PATH as one colon-separated element; compare what child
# processes see.
set -l orig_path (string join : $PATH)
set -gx JAVA_HOME /orig
set -e JAVA_HOME_BEFORE_JAVM
set -e JAVM_ACTIVE_JAVA_HOME
javm use 21; or exit 1
if test "$JAVA_HOME" = /orig
    echo "use did not change JAVA_HOME"; exit 1
end
if test "$JAVA_HOME_BEFORE_JAVM" != /orig
    echo "JAVA_HOME_BEFORE_JAVM=[$JAVA_HOME_BEFORE_JAVM], want [/orig]"; exit 1
end
javm deactivate; or exit 1
if test "$JAVA_HOME" != /orig
    echo "JAVA_HOME=[$JAVA_HOME] after deactivate, want [/orig]"; exit 1
end
if set -q JAVA_HOME_BEFORE_JAVM
    echo "JAVA_HOME_BEFORE_JAVM is still set"; exit 1
end

# A discovered JDK lives outside JAVM_HOME; its bin must not stay in PATH.
javm use testvendor-gradle@17; or exit 1
javm use testvendor-gradle@17; or exit 1
if test (javm current) != testvendor-gradle@17
    echo "current=["(javm current)"], want [testvendor-gradle@17]"; exit 1
end
javm deactivate; or exit 1
# The real PATH may have a system java that discovery knows; hide it.
PATH=/nonexistent $JAVM_BINARY current >/dev/null 2>&1
set -l rc $status
if test $rc -ne 3
    echo "current exited with $rc after deactivate, want 3"; exit 1
end
if test (string join : $PATH) != "$orig_path"
    echo "PATH=["(string join : $PATH)"] after deactivate, want [$orig_path]"; exit 1
end
if test "$JAVA_HOME" != /orig
    echo "JAVA_HOME=[$JAVA_HOME] after deactivate, want [/orig]"; exit 1
end
if set -q JAVM_ACTIVE_JAVA_HOME
    echo "JAVM_ACTIVE_JAVA_HOME is still set"; exit 1
end

# Without an original JAVA_HOME, deactivate unsets it instead of leaving it empty.
set -e JAVA_HOME
javm use 21; or exit 1
javm use testvendor-gradle@17; or exit 1
javm deactivate; or exit 1
if set -q JAVA_HOME
    echo "JAVA_HOME=[$JAVA_HOME] after deactivate, want unset"; exit 1
end
if test (string join : $PATH) != "$orig_path"
    echo "PATH=["(string join : $PATH)"] after deactivate, want [$orig_path]"; exit 1
end
`
