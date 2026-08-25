package command

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	osExec "os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/felipebz/javm/discovery"
)

func TestParseExecArgsRequiresDashAndCommand(t *testing.T) {
	cmd := NewExecCommand()
	cmd.SetArgs([]string{"21", "java"})
	if err := cmd.ValidateArgs([]string{"21", "java"}); !errors.Is(err, ErrUsage) {
		t.Fatalf("ValidateArgs() = %v, want usage error", err)
	}

	cmd = NewExecCommand()
	cmd.SetArgs([]string{"21", "--"})
	if err := cmd.Execute(); !errors.Is(err, ErrUsage) || !strings.Contains(err.Error(), "no command specified") {
		t.Fatalf("Execute() = %v, want clear missing-command usage error", err)
	}
}

func TestResolveJDKContextSupportsSelectorAndAlias(t *testing.T) {
	home := t.TempDir()
	t.Setenv("JAVM_HOME", home)
	cleanup := setupMockLs()
	defer cleanup()

	selectedPath := filepath.Join(home, "jdk", "temurin@21.0.1")
	mockLsResult = []discovery.JDK{{
		Identifier: "temurin@21.0.1",
		Version:    "21.0.1",
		Source:     "javm",
		Path:       selectedPath,
	}}

	resolved, err := resolveJDKContext(context.Background(), "21")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Identifier != "temurin@21.0.1" {
		t.Fatalf("explicit selector resolved to %q", resolved.Identifier)
	}

	if err := os.WriteFile(filepath.Join(home, "team.alias"), []byte("temurin@21"), 0o600); err != nil {
		t.Fatal(err)
	}
	resolved, err = resolveJDKContext(context.Background(), "team")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Identifier != "temurin@21.0.1" {
		t.Fatalf("alias resolved to %q", resolved.Identifier)
	}
}

func TestExecResolutionErrorsAreClassified(t *testing.T) {
	home := t.TempDir()
	t.Setenv("JAVM_HOME", home)
	cleanup := setupMockLs()
	defer cleanup()
	mockLsResult = []discovery.JDK{{
		Identifier: "system@21.0.1",
		Version:    "21.0.1",
		Source:     "javm",
		Path:       filepath.Join(home, "jdk", "system@21.0.1"),
	}}

	if _, err := resolveJDKContext(context.Background(), "not a selector"); !errors.Is(err, ErrUsage) {
		t.Fatalf("invalid selector error = %v, want usage error", err)
	}
	if _, err := resolveJDKContext(context.Background(), "99"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing selector error = %v, want not-found error", err)
	}
}

func TestExecUsesJavaVersionWhenSelectorIsOmitted(t *testing.T) {
	home := t.TempDir()
	workspace := t.TempDir()
	t.Setenv("JAVM_HOME", home)
	if err := os.WriteFile(filepath.Join(workspace, ".java-version"), []byte("system@21\n"), 0o600); err != nil {
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

	cleanup := setupMockLs()
	defer cleanup()
	mockLsResult = []discovery.JDK{{
		Identifier: "system@21.0.1",
		Version:    "21.0.1",
		Source:     "javm",
		Path:       filepath.Join(home, "jdk", "system@21.0.1"),
	}}

	selector, childArgs, err := parseExecArgsForTest([]string{"--", "java", "--version"})
	if err != nil {
		t.Fatal(err)
	}
	if selector != "system@21" || fmt.Sprint(childArgs) != "[java --version]" {
		t.Fatalf("omitted selector parsed as %q %v", selector, childArgs)
	}
}

func parseExecArgsForTest(args []string) (string, []string, error) {
	cmd := NewExecCommand()
	cmd.SetArgs(args)
	if err := cmd.ParseFlags(args); err != nil {
		return "", nil, err
	}
	return parseExecArgs(cmd, cmd.Flags().Args())
}

func TestBuildJDKEnvironmentDoesNotMutateParent(t *testing.T) {
	home := t.TempDir()
	oldManaged := filepath.Join(home, "jdk", "old", "bin")
	oldPath := strings.Join([]string{oldManaged, "/outside/bin"}, string(os.PathListSeparator))
	selectedPath := filepath.Join(home, "jdk", "new")
	t.Setenv("JAVM_HOME", home)
	t.Setenv("PATH", oldPath)
	t.Setenv("JAVA_HOME", "/parent/java")

	envBefore := os.Environ()
	selected, err := buildJDKEnvironment(selectedPath, envBefore)
	if err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("PATH"); got != oldPath {
		t.Fatalf("parent PATH changed to %q", got)
	}
	if got := os.Getenv("JAVA_HOME"); got != "/parent/java" {
		t.Fatalf("parent JAVA_HOME changed to %q", got)
	}

	wantPrefix := filepath.Join(jdkHomeForTest(selectedPath), "bin") + string(os.PathListSeparator)
	if !strings.HasPrefix(selected.Path, wantPrefix) {
		t.Fatalf("selected PATH = %q, want prefix %q", selected.Path, wantPrefix)
	}
	if strings.Contains(selected.Path, oldManaged) {
		t.Fatalf("selected PATH retained old managed entry: %q", selected.Path)
	}
}

func TestBuildJDKEnvironmentDoesNotCreateCurrentDirectoryEntry(t *testing.T) {
	home := t.TempDir()
	oldManaged := filepath.Join(home, "jdk", "old", "bin")
	selectedPath := filepath.Join(home, "jdk", "new")
	t.Setenv("JAVM_HOME", home)

	selected, err := buildJDKEnvironment(selectedPath, []string{"PATH=" + oldManaged})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(jdkHomeForTest(selectedPath), "bin")
	if selected.Path != want {
		t.Fatalf("selected PATH = %q, want %q", selected.Path, want)
	}

	selected, err = buildJDKEnvironment(selectedPath, []string{"PATH=" + oldManaged + string(os.PathListSeparator)})
	if err != nil {
		t.Fatal(err)
	}
	want = filepath.Join(selectedPath, "bin") + string(os.PathListSeparator)
	if selected.Path != want {
		t.Fatalf("selected PATH with original empty entry = %q, want %q", selected.Path, want)
	}

	selected, err = buildJDKEnvironment(selectedPath, []string{"PATH="})
	if err != nil {
		t.Fatal(err)
	}
	if selected.Path != want {
		t.Fatalf("selected PATH with explicitly empty value = %q, want %q", selected.Path, want)
	}
}

func TestFindExecutableReportsPermissionDenied(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not use Unix executable permission bits")
	}

	dir := t.TempDir()
	tool := filepath.Join(dir, "tool")
	if err := os.WriteFile(tool, []byte("not executable"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := findExecutableInEnvironment("tool", []string{"PATH=" + dir}); !errors.Is(err, os.ErrPermission) {
		t.Fatalf("findExecutableInEnvironment() = %v, want permission error", err)
	}
}

func TestExecFindsExecutableUsingChildPathAndPreservesInvocation(t *testing.T) {
	if runtime.GOOS == "plan9" {
		t.Skip("test helper is only supported on the target platforms")
	}

	home := t.TempDir()
	selectedRoot := filepath.Join(home, "jdk", "temurin@21.0.1")
	wrongRoot := filepath.Join(t.TempDir(), "other-jdk")
	selectedBin := filepath.Join(jdkHomeForTest(selectedRoot), "bin")
	wrongBin := filepath.Join(jdkHomeForTest(wrongRoot), "bin")
	if err := os.MkdirAll(selectedBin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(wrongBin, 0o755); err != nil {
		t.Fatal(err)
	}

	selectedProbe := filepath.Join(selectedBin, executableName("probe"))
	wrongProbe := filepath.Join(wrongBin, executableName("probe"))
	buildExecHelper(t, selectedProbe, "selected")
	buildExecHelper(t, wrongProbe, "wrong")

	workspace := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(workspace); err != nil {
		t.Fatal(err)
	}
	expectedWorkingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	t.Setenv("JAVM_HOME", home)
	t.Setenv("PATH", wrongBin)
	t.Setenv("JAVA_HOME", "/parent/java")
	t.Setenv("JAVM_TEST_EXIT", "")
	cleanup := setupMockLs()
	defer cleanup()
	mockLsResult = []discovery.JDK{{
		Identifier: "temurin@21.0.1",
		Version:    "21.0.1",
		Source:     "javm",
		Path:       selectedRoot,
	}}

	var out, errOut bytes.Buffer
	cmd := NewExecCommand()
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs([]string{"21", "--", "probe", "--foo", "value with spaces", "--bar=baz"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("exec failed: %v\nstderr: %s", err, errOut.String())
	}

	output := out.String()
	if !strings.Contains(output, "LABEL=selected\n") {
		t.Fatalf("child was not selected from child PATH:\n%s", output)
	}
	if !strings.Contains(output, "JAVA_HOME="+jdkHomeForTest(selectedRoot)+"\n") {
		t.Fatalf("child JAVA_HOME is wrong:\n%s", output)
	}
	wantChildPath := selectedBin + string(os.PathListSeparator) + wrongBin
	if !strings.Contains(output, "PATH="+wantChildPath+"\n") {
		t.Fatalf("child PATH is wrong:\n%s", output)
	}
	if !strings.Contains(output, "ARGS=[\"--foo\" \"value with spaces\" \"--bar=baz\"]\n") {
		t.Fatalf("child arguments were not preserved:\n%s", output)
	}
	if !strings.Contains(output, "PWD="+expectedWorkingDirectory+"\n") {
		t.Fatalf("child working directory changed:\n%s", output)
	}
	if got := os.Getenv("PATH"); got != wrongBin {
		t.Fatalf("parent PATH changed to %q", got)
	}
	if got := os.Getenv("JAVA_HOME"); got != "/parent/java" {
		t.Fatalf("parent JAVA_HOME changed to %q", got)
	}
}

func TestExecPropagatesChildExitCode(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(home, "jdk", "temurin@21.0.1")
	bin := filepath.Join(jdkHomeForTest(root), "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	probe := filepath.Join(bin, executableName("probe"))
	buildExecHelper(t, probe, "selected")

	t.Setenv("JAVM_HOME", home)
	t.Setenv("PATH", bin)
	t.Setenv("JAVM_TEST_EXIT", "7")
	cleanup := setupMockLs()
	defer cleanup()
	mockLsResult = []discovery.JDK{{Identifier: "temurin@21.0.1", Version: "21.0.1", Source: "javm", Path: root}}

	cmd := NewExecCommand()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetArgs([]string{"21", "--", "probe"})
	err := cmd.Execute()
	var exitErr *osExec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("exec error = %v, want child exit error", err)
	}
	if got := exitErr.ExitCode(); got != 7 {
		t.Fatalf("child exit code = %d, want 7", got)
	}
}

func TestExecReportsMissingChildCommand(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(home, "jdk", "temurin@21.0.1")
	if err := os.MkdirAll(filepath.Join(jdkHomeForTest(root), "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("JAVM_HOME", home)
	t.Setenv("PATH", filepath.Join(jdkHomeForTest(root), "bin"))
	cleanup := setupMockLs()
	defer cleanup()
	mockLsResult = []discovery.JDK{{Identifier: "temurin@21.0.1", Version: "21.0.1", Source: "javm", Path: root}}

	cmd := NewExecCommand()
	cmd.SetArgs([]string{"21", "--", "definitely-not-installed"})
	err := cmd.Execute()
	if !errors.Is(err, ErrNotFound) || !strings.Contains(err.Error(), `command "definitely-not-installed" was not found`) {
		t.Fatalf("missing child command error = %v", err)
	}
}

func TestExecReportsInaccessibleSelectedJDK(t *testing.T) {
	home := t.TempDir()
	t.Setenv("JAVM_HOME", home)
	t.Setenv("PATH", "")
	cleanup := setupMockLs()
	defer cleanup()
	mockLsResult = []discovery.JDK{{
		Identifier: "system@21.0.1",
		Version:    "21.0.1",
		Source:     "javm",
		Path:       filepath.Join(home, "external-jdk-that-was-removed"),
	}}

	cmd := NewExecCommand()
	cmd.SetArgs([]string{"system@21", "--", "java"})
	err := cmd.Execute()
	if !errors.Is(err, os.ErrNotExist) || !strings.Contains(err.Error(), `selected JDK "system@21.0.1" is inaccessible`) {
		t.Fatalf("inaccessible JDK error = %v", err)
	}
}

func jdkHomeForTest(root string) string {
	if runtime.GOOS == "darwin" {
		return filepath.Join(root, "Contents", "Home")
	}
	return root
}

func executableName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

func buildExecHelper(t *testing.T, output, label string) {
	t.Helper()
	packageDir := filepath.Dir(mustCurrentTestFile(t))
	build := osExec.Command("go", "build", "-ldflags", "-X=main.label="+label, "-o", output, "./testdata/exec-helper")
	build.Dir = packageDir
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build exec helper: %v\n%s", err, output)
	}
}

func mustCurrentTestFile(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(1)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return file
}
