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

func TestRootExecPassesDashArgumentsOpaque(t *testing.T) {
	probe := createRootExecFixture(t)

	tests := []struct {
		name            string
		prefix          []string
		beforeSeparator []string
		selector        string
		arguments       []string
	}{
		{name: "implicit Maven profile", arguments: []string{"-Pfrontend-api-client", "package"}},
		{name: "implicit system property", arguments: []string{"-Dfoo=bar"}},
		{name: "implicit stacktrace", arguments: []string{"--stacktrace"}},
		{name: "implicit JVM option", arguments: []string{"-Xmx2g"}},
		{name: "explicit selector", selector: "21", arguments: []string{"-Pfrontend-api-client"}},
		{name: "root persistent flag before command", prefix: []string{"--quiet"}, arguments: []string{"-Pfrontend-api-client"}},
		{name: "root persistent flag after command", beforeSeparator: []string{"--quiet"}, arguments: []string{"-Pfrontend-api-client"}},
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
				args = append(args, tt.selector)
			}
			args = append(args, tt.beforeSeparator...)
			args = append(args, "--")
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
