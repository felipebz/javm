package command

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felipebz/javm/discovery"
)

// linkedAndDiscovered returns a JDK linked as system@21.0.4 and found by the
// intellij source at its real directory, in discovery order.
func linkedAndDiscovered(t *testing.T) []discovery.JDK {
	t.Helper()
	skipWithoutSymlinks(t)
	t.Setenv("JAVM_HOME", t.TempDir())
	real := filepath.Join(t.TempDir(), "jdks", "temurin-21.0.4")
	if err := os.MkdirAll(real, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(os.Getenv("JAVM_HOME"), "jdk", "system@21.0.4")
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	return []discovery.JDK{
		{Identifier: "system@21.0.4", Version: "21.0.4", Vendor: "Eclipse Adoptium", Architecture: "x64", Source: "javm", Path: link},
		{Identifier: "eclipse-adoptium-intellij@21", Version: "21.0.4", Vendor: "Eclipse Adoptium", Architecture: "x64", Source: "intellij", Path: real},
	}
}

func runLs(t *testing.T, args ...string) string {
	t.Helper()
	cmd := NewLsCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs(append([]string{}, args...))
	if err := cmd.Execute(); err != nil {
		t.Fatalf("ls %v: %v", args, err)
	}
	return out.String()
}

func TestLsShowsLinkedAndDiscoveredJDKOnce(t *testing.T) {
	t.Cleanup(setupMockLs())
	mockLsResult = linkedAndDiscovered(t)

	out := runLs(t)
	rows := parseLsOutput(t, out)
	if len(rows) != 1 || rows[0].name != "system@21.0.4" || rows[0].source != "javm" {
		t.Fatalf("ls rows = %+v, want one system@21.0.4 row:\n%s", rows, out)
	}
	if !listsName(rows[0].selectedBy, "eclipse-adoptium-intellij@21") {
		t.Fatalf("SELECTED BY = %q, want the other identifier listed:\n%s", rows[0].selectedBy, out)
	}

	details := runLs(t, "--details")
	for _, want := range []string{mockLsResult[0].Path, mockLsResult[1].Path, "intellij", "eclipse-adoptium-intellij@21"} {
		if !strings.Contains(details, want) {
			t.Fatalf("ls --details misses %q:\n%s", want, details)
		}
	}
	if got := strings.Count(strings.TrimSpace(details), "\n"); got != 2 {
		t.Fatalf("ls --details has %d rows, want the two sources of one JDK:\n%s", got, details)
	}
}

func TestLsRangeKeepsGroupedJDK(t *testing.T) {
	t.Cleanup(setupMockLs())
	mockLsResult = linkedAndDiscovered(t)

	for _, rng := range []string{"21", "eclipse-adoptium-intellij@21", "system@21"} {
		rows := parseLsOutput(t, runLs(t, rng))
		if len(rows) != 1 || rows[0].name != "system@21.0.4" {
			t.Fatalf("ls %s rows = %+v, want the grouped JDK once", rng, rows)
		}
	}
}

func TestLsGroupsMacOSBundleAndContentsHome(t *testing.T) {
	t.Cleanup(setupMockLs())
	forceGOOS(t, "darwin")
	bundle := filepath.Join(t.TempDir(), "temurin-21.jdk")
	home := filepath.Join(bundle, "Contents", "Home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	mockLsResult = []discovery.JDK{
		{Identifier: "eclipse-adoptium-system@21", Version: "21.0.4", Source: "system", Path: bundle},
		{Identifier: "eclipse-adoptium-gradle@21", Version: "21.0.4", Source: "gradle", Path: home},
	}

	out := runLs(t)
	rows := parseLsOutput(t, out)
	if len(rows) != 1 || rows[0].name != "eclipse-adoptium-system@21" || !listsName(rows[0].selectedBy, "eclipse-adoptium-gradle@21") {
		t.Fatalf("ls rows = %+v, want one row naming the other identifier:\n%s", rows, out)
	}
}

func TestLsPrefersManagedJDKAndListsLinkName(t *testing.T) {
	t.Cleanup(setupMockLs())
	skipWithoutSymlinks(t)
	t.Setenv("JAVM_HOME", t.TempDir())
	managed := filepath.Join(os.Getenv("JAVM_HOME"), "jdk", "temurin@21.0.4")
	if err := os.MkdirAll(managed, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(os.Getenv("JAVM_HOME"), "jdk", "system@21.0.4")
	if err := os.Symlink(managed, link); err != nil {
		t.Fatal(err)
	}
	mockLsResult = []discovery.JDK{
		{Identifier: "system@21.0.4", Version: "21.0.4", Source: "javm", Path: link},
		{Identifier: "temurin@21.0.4", Version: "21.0.4", Source: "javm", Path: managed},
	}

	out := runLs(t)
	rows := parseLsOutput(t, out)
	if len(rows) != 1 || rows[0].name != "temurin@21.0.4" || !strings.Contains(rows[0].selectedBy, "also system@21.0.4") {
		t.Fatalf("ls rows = %+v, want temurin@21.0.4 also naming system@21.0.4:\n%s", rows, out)
	}
}

// listsName reports whether SELECTED BY names name, as a selector or after
// "also".
func listsName(selectedBy, name string) bool {
	for field := range strings.FieldsFuncSeq(selectedBy, func(r rune) bool { return r == ',' || r == ';' || r == ' ' }) {
		if field == name {
			return true
		}
	}
	return false
}

func TestLsKeepsDifferentJDKsWithSameVersion(t *testing.T) {
	t.Cleanup(setupMockLs())
	mockLsResult = []discovery.JDK{
		{Identifier: "eclipse-adoptium-system@21", Version: "21.0.4", Source: "system", Path: filepath.Join(t.TempDir(), "a")},
		{Identifier: "eclipse-adoptium-intellij@21", Version: "21.0.4", Source: "intellij", Path: filepath.Join(t.TempDir(), "b")},
	}

	if rows := parseLsOutput(t, runLs(t)); len(rows) != 2 {
		t.Fatalf("ls rows = %+v, want two different JDKs", rows)
	}
}

func TestGroupedJDKIdentifiersStillResolve(t *testing.T) {
	t.Cleanup(setupMockLs())
	mockLsResult = linkedAndDiscovered(t)

	for _, jdk := range mockLsResult {
		cmd := NewWhichCommand()
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{jdk.Identifier})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("which %s: %v", jdk.Identifier, err)
		}
		if got := strings.TrimSpace(out.String()); got != jdk.Path {
			t.Fatalf("which %s = %q, want %q", jdk.Identifier, got, jdk.Path)
		}
	}
}
