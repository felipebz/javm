package command

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felipebz/javm/discovery"
)

type fakeJavaHomeRunner struct {
	home  string
	err   error
	calls int
}

func (r *fakeJavaHomeRunner) CombinedOutput(context.Context, string, ...string) ([]byte, error) {
	r.calls++
	if r.err != nil {
		return []byte("Unable to locate a Java Runtime.\n"), r.err
	}
	return []byte(r.home + "\n"), nil
}

// macOSLauncherOnPath simulates macOS with the launcher stub as the java on
// PATH and java_home answering through runner.
func macOSLauncherOnPath(t *testing.T, runner *fakeJavaHomeRunner) {
	t.Helper()
	isolateCurrentEnvironment(t)
	forceGOOS(t, "darwin")
	stub := filepath.Join(t.TempDir(), "java")
	if err := os.WriteFile(stub, []byte("stub"), 0o755); err != nil {
		t.Fatal(err)
	}
	stub, err := filepath.EvalSymlinks(stub)
	if err != nil {
		t.Fatal(err)
	}
	originalLauncher, originalRunner := macOSJavaLauncher, javaHomeRunner
	macOSJavaLauncher, javaHomeRunner = stub, runner
	t.Cleanup(func() { macOSJavaLauncher, javaHomeRunner = originalLauncher, originalRunner })
	lookPath = func(string) (string, error) { return stub, nil }
}

// macOSBundle creates root/Contents/Home/bin/java and returns the home.
func macOSBundle(t *testing.T, root string) string {
	t.Helper()
	home := filepath.Join(root, "Contents", "Home")
	if err := os.MkdirAll(filepath.Join(home, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "bin", "java"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return home
}

func TestCurrentReportsJDKRunByMacOSLauncher(t *testing.T) {
	runner := &fakeJavaHomeRunner{}
	macOSLauncherOnPath(t, runner)
	root := filepath.Join(t.TempDir(), "temurin-21.jdk")
	runner.home = macOSBundle(t, root)
	mockLsResult = []discovery.JDK{{Identifier: "temurin@21", Source: "system", Path: root}}

	out, err := runCurrent(t)
	if err != nil || strings.TrimSpace(out) != "temurin@21" {
		t.Fatalf("current = %q, %v; want temurin@21", out, err)
	}
}

func TestCurrentPrefersJavaHomeOverJavaHomeTool(t *testing.T) {
	runner := &fakeJavaHomeRunner{}
	macOSLauncherOnPath(t, runner)
	system := filepath.Join(t.TempDir(), "temurin-25.jdk")
	runner.home = macOSBundle(t, system)
	managed := filepath.Join(t.TempDir(), "zulu@21.0.4")
	t.Setenv("JAVA_HOME", macOSBundle(t, managed))
	mockLsResult = []discovery.JDK{
		{Identifier: "temurin@25", Source: "system", Path: system},
		{Identifier: "zulu@21.0.4", Source: "javm", Path: managed},
	}

	out, err := runCurrent(t)
	if err != nil || strings.TrimSpace(out) != "zulu@21.0.4" {
		t.Fatalf("current = %q, %v; want zulu@21.0.4", out, err)
	}
	if runner.calls != 0 {
		t.Fatalf("java_home ran %d times although JAVA_HOME holds a JDK", runner.calls)
	}
}

func TestCurrentReportsMacOSLauncherWithoutJDK(t *testing.T) {
	macOSLauncherOnPath(t, &fakeJavaHomeRunner{err: errors.New("exit status 1")})
	mockLsResult = []discovery.JDK{}

	_, err := runCurrent(t)
	if !errors.Is(err, ErrNotFound) || !strings.Contains(err.Error(), "macOS launcher") || !strings.Contains(err.Error(), "no JDK to run") {
		t.Fatalf("current error = %v, want not found naming the launcher without a JDK", err)
	}
}

func TestCurrentNamesUnknownJDKRunByMacOSLauncher(t *testing.T) {
	runner := &fakeJavaHomeRunner{}
	macOSLauncherOnPath(t, runner)
	runner.home = macOSBundle(t, filepath.Join(t.TempDir(), "unknown.jdk"))
	mockLsResult = []discovery.JDK{}

	_, err := runCurrent(t)
	if !errors.Is(err, ErrNotFound) || !strings.Contains(err.Error(), "macOS launcher") || !strings.Contains(err.Error(), runner.home) {
		t.Fatalf("current error = %v, want not found naming %s", err, runner.home)
	}
}
