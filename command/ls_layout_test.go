package command

import (
	"bytes"
	"strings"
	"testing"

	"github.com/felipebz/javm/discovery"
	"github.com/muesli/termenv"
)

// smokeTestJDKs reproduces the misaligned smoke test table: a managed JDK
// without selectors between rows that have them.
func smokeTestJDKs() []discovery.JDK {
	return []discovery.JDK{
		{Identifier: "temurin@21.0.5", Version: "21.0.5", Source: "javm", Path: "/jdks/temurin@21.0.5"},
		{Identifier: "temurin@21.0.4", Version: "21.0.4", Source: "javm", Path: "/jdks/temurin@21.0.4"},
		{Identifier: "temurin@17.0.13", Version: "17.0.13", Source: "javm", Path: "/jdks/temurin@17.0.13"},
		{Identifier: "amazon-com-inc-intellij@21", Version: "21.0.2", Source: "intellij", Path: "/home/.jdks/corretto-21"},
		{Identifier: "arch-linux-system@21", Version: "21.0.9", Source: "system", Path: "/usr/lib/jvm/java-21"},
	}
}

func printLs(t *testing.T, jdks []discovery.JDK, details bool, opts ...termenv.OutputOption) string {
	t.Helper()
	var out bytes.Buffer
	if len(opts) == 0 {
		opts = []termenv.OutputOption{termenv.WithProfile(termenv.Ascii)}
	}
	if err := printInstalledVersions(&out, newStyles(&out, opts...), jdks, nil, details); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

// assertColumnsAligned checks that every value of a header column starts at
// that column's offset on each line where the line has text there.
func assertColumnsAligned(t *testing.T, table string, columns ...string) {
	t.Helper()
	lines := strings.Split(strings.TrimSuffix(table, "\n"), "\n")
	starts := map[int]bool{}
	for _, column := range columns {
		starts[strings.Index(lines[0], column)] = true
	}
	starts[0] = true
	for _, line := range lines[1:] {
		// A cell starts at the line start or after two or more spaces; group
		// member rows indent SOURCE and NAME by two spaces.
		for i := range len(line) {
			if line[i] == ' ' || (i > 0 && line[i-1] != ' ') || (i > 1 && line[i-2] != ' ') {
				continue
			}
			if !starts[i] && !(strings.HasPrefix(line, "  ") && starts[i-2]) {
				t.Fatalf("a cell starts at offset %d, not at a column, in %q:\n%s", i, line, table)
			}
		}
	}
	for _, line := range lines {
		if strings.HasSuffix(line, " ") {
			t.Fatalf("line %q ends with a space:\n%s", line, table)
		}
	}
}

func TestLsColumnsAlignWhenSelectedByIsEmpty(t *testing.T) {
	assertColumnsAligned(t, printLs(t, smokeTestJDKs(), false), "SOURCE", "SELECTED BY")
}

func TestLsStyledLayoutMatchesPlain(t *testing.T) {
	plain := printLs(t, smokeTestJDKs(), false)
	styled := printLs(t, smokeTestJDKs(), false, termenv.WithProfile(termenv.ANSI))
	if got := ansiEscape.ReplaceAllString(styled, ""); got != plain {
		t.Fatalf("styled output without escapes:\n%s\nwant the plain layout:\n%s", got, plain)
	}
	accent := newStyles(&bytes.Buffer{}, termenv.WithProfile(termenv.ANSI)).accent
	if !strings.Contains(styled, accent.Styled("21, temurin@21")) {
		t.Fatalf("SELECTED BY is not styled as one accent cell:\n%q", styled)
	}
	if strings.Contains(styled, accent.Styled("")) {
		t.Fatalf("an empty SELECTED BY cell was styled:\n%q", styled)
	}
}

func TestLsDetailsColumnsAlignWithGroupsAndEmptyCells(t *testing.T) {
	jdks := smokeTestJDKs()
	jdks = append(jdks, discovery.JDK{Identifier: "temurin-gradle@21", Version: "21.0.4", Source: "gradle", Path: jdks[1].Path})
	assertColumnsAligned(t, printLs(t, jdks, true), "NAME", "SELECTED BY", "VENDOR", "ARCHITECTURE", "PATH")
}

func TestLsFullTableLayoutUnchanged(t *testing.T) {
	jdks := []discovery.JDK{
		{Identifier: "temurin@21.0.5", Version: "21.0.5", Source: "javm", Path: "/a"},
		{Identifier: "temurin@17.0.13", Version: "17.0.13", Source: "javm", Path: "/b"},
	}
	want := "NAME              SOURCE   SELECTED BY\n" +
		"temurin@21.0.5    javm     21, temurin@21\n" +
		"temurin@17.0.13   javm     17, temurin@17\n"
	if got := printLs(t, jdks, false); got != want {
		t.Fatalf("ls =\n%q\nwant\n%q", got, want)
	}
}
