package main

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/felipebz/javm/discoapi"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const legacyHint = `(Java 8 and earlier are written without "1.", for example 8)`

// writeFakeJava8 creates a managed Temurin 8 whose release file reports the
// legacy 1.8.0_502 version, as real Java 8 builds do.
func writeFakeJava8(t *testing.T, home string) {
	t.Helper()
	jdkHome := filepath.Join(home, "jdk", "temurin@8.0.502")
	if runtime.GOOS == "darwin" {
		jdkHome = filepath.Join(jdkHome, "Contents", "Home")
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
	release := "JAVA_VERSION=\"1.8.0_502\"\nJAVA_VENDOR=\"TestVendor\"\nOS_ARCH=\"x64\"\n"
	if err := os.WriteFile(filepath.Join(jdkHome, "release"), []byte(release), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestWhichLegacySelectorExplainsModernVersions(t *testing.T) {
	_, home := isolatedWorkspace(t)
	writeFakeJava8(t, home)

	if stdout, stderr, code := executeRoot(t, "which", "8"); code != 0 || stdout == "" {
		t.Fatalf("which 8: exit %d, stdout %q, stderr %q, want the Java 8 fixture", code, stdout, stderr)
	}
	stdout, stderr, code := executeRoot(t, "which", "1.8")
	if code != exitNotFound || stdout != "" {
		t.Fatalf("which 1.8: exit %d, stdout %q, stderr %q, want exit %d and no output", code, stdout, stderr, exitNotFound)
	}
	if want := "1.8 isn't installed " + legacyHint; !strings.Contains(stderr, want) {
		t.Fatalf("which 1.8: stderr %q, want %q", stderr, want)
	}
}

func TestWhichModernSelectorHasNoLegacyHint(t *testing.T) {
	_, home := isolatedWorkspace(t)
	writeFakeJava8(t, home)

	stdout, stderr, code := executeRoot(t, "which", "99")
	if code != exitNotFound || stdout != "" {
		t.Fatalf("which 99: exit %d, stdout %q, stderr %q, want exit %d and no output", code, stdout, stderr, exitNotFound)
	}
	if !strings.Contains(stderr, "99 isn't installed") || strings.Contains(stderr, `"1."`) {
		t.Fatalf("which 99: stderr %q, want the not-found error without the legacy hint", stderr)
	}
}

func TestHelpExamplesUseModernVersions(t *testing.T) {
	root := newRootCommand(application{
		out:    io.Discard,
		err:    io.Discard,
		logger: log.New(),
		client: discoapi.NewClient(),
	})
	var walk func(*cobra.Command)
	walk = func(cmd *cobra.Command) {
		if strings.Contains(cmd.Example, "1.8") {
			t.Errorf("%q example uses a legacy 1.x version:\n%s", cmd.CommandPath(), cmd.Example)
		}
		for _, child := range cmd.Commands() {
			walk(child)
		}
	}
	walk(root)
}
