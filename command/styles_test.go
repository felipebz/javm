package command

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"github.com/muesli/termenv"
)

var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)

// stripANSI returns the output as a terminal without color support would show
// it, which is what the table alignment has to match.
func stripANSI(s string) string {
	return ansiEscape.ReplaceAllString(s, "")
}

// clearColorEnv removes the color variables the developer or CI may have set,
// so the tests do not depend on the environment where go test runs.
func clearColorEnv(t *testing.T) {
	t.Helper()
	t.Setenv("NO_COLOR", "")
	t.Setenv("CLICOLOR", "")
	t.Setenv("CLICOLOR_FORCE", "")
	t.Setenv("TERM", "")
	t.Setenv("COLORTERM", "")
}

func TestStylesStayPlainForNonTerminalWriter(t *testing.T) {
	clearColorEnv(t)

	styles := newStyles(&bytes.Buffer{})
	for _, value := range []string{styles.Header("NAME"), styles.Accent("25, temurin@25"), styles.Dim("javm")} {
		if strings.Contains(value, "\x1b") {
			t.Errorf("unstyled writer produced escapes: %q", value)
		}
	}
	if got := styles.Accent(""); got != "" {
		t.Errorf("empty value = %q, want no escape sequence", got)
	}
}

// enableANSIConsole is best-effort: it must always return something callable and
// never keep a writer that cannot be configured for ANSI from working.
func TestEnableANSIConsoleIsBestEffort(t *testing.T) {
	clearColorEnv(t)

	restore := enableANSIConsole(&bytes.Buffer{})
	if restore == nil {
		t.Fatal("enableANSIConsole returned no restore function")
	}
	restore()
}

func TestStylesFollowColorOverrides(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T)
		opts  []termenv.OutputOption
		want  bool
	}{
		{
			name:  "interactive terminal",
			setup: func(t *testing.T) { clearColorEnv(t); t.Setenv("TERM", "xterm-256color") },
			opts:  []termenv.OutputOption{termenv.WithTTY(true)},
			want:  true,
		},
		{
			name:  "NO_COLOR",
			setup: func(t *testing.T) { clearColorEnv(t); t.Setenv("TERM", "xterm-256color"); t.Setenv("NO_COLOR", "1") },
			opts:  []termenv.OutputOption{termenv.WithTTY(true)},
			want:  false,
		},
		{
			name:  "NO_COLOR wins over CLICOLOR_FORCE",
			setup: func(t *testing.T) { clearColorEnv(t); t.Setenv("NO_COLOR", "1"); t.Setenv("CLICOLOR_FORCE", "1") },
			opts:  []termenv.OutputOption{termenv.WithTTY(true)},
			want:  false,
		},
		{
			name:  "CLICOLOR_FORCE on a redirected output",
			setup: func(t *testing.T) { clearColorEnv(t); t.Setenv("CLICOLOR_FORCE", "1") },
			want:  true,
		},
		{
			name:  "CLICOLOR=0 on an interactive terminal",
			setup: func(t *testing.T) { clearColorEnv(t); t.Setenv("TERM", "xterm-256color"); t.Setenv("CLICOLOR", "0") },
			opts:  []termenv.OutputOption{termenv.WithTTY(true)},
			want:  false,
		},
		{
			name:  "terminal without color support",
			setup: func(t *testing.T) { clearColorEnv(t); t.Setenv("TERM", "dumb") },
			opts:  []termenv.OutputOption{termenv.WithTTY(true)},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup(t)

			var out bytes.Buffer
			styles := newStyles(&out, tt.opts...)
			got := styles.Accent("25, temurin@25")

			if strings.Contains(got, "\x1b") != tt.want {
				t.Errorf("Accent() = %q, want styling %v", got, tt.want)
			}
			if plain := stripANSI(got); plain != "25, temurin@25" {
				t.Errorf("visible value = %q, want %q", plain, "25, temurin@25")
			}
		})
	}
}

func TestStylesApplyHierarchyForForcedProfile(t *testing.T) {
	clearColorEnv(t)

	styles := newStyles(&bytes.Buffer{}, termenv.WithProfile(termenv.TrueColor))

	t.Run("header is bold and keeps the terminal color", func(t *testing.T) {
		got := styles.Header("SELECTED BY")
		if !strings.Contains(got, termenv.CSI+termenv.BoldSeq+"m") {
			t.Errorf("header %q is not bold", got)
		}
		if strings.Contains(got, "38;") {
			t.Errorf("header %q is colored, want the default terminal color", got)
		}
		if plain := stripANSI(got); plain != "SELECTED BY" {
			t.Errorf("header text = %q, want %q", plain, "SELECTED BY")
		}
	})

	t.Run("accent value uses the javm color", func(t *testing.T) {
		want := "\x1b[38;2;216;84;10mliberica@25\x1b[0m"
		if got := styles.Accent("liberica@25"); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("dim value is faint and uncolored", func(t *testing.T) {
		got := styles.Dim("javm")
		if !strings.Contains(got, termenv.CSI+termenv.FaintSeq+"m") {
			t.Errorf("value %q is not dimmed", got)
		}
		if strings.Contains(got, "38;") {
			t.Errorf("dim value %q is colored, want a color independent dim", got)
		}
		if plain := stripANSI(got); plain != "javm" {
			t.Errorf("dim text = %q, want %q", plain, "javm")
		}
	})
}

func TestStylesFollowTerminalColorSupport(t *testing.T) {
	clearColorEnv(t)

	const value = "25, temurin@25"

	// TrueColor is the only profile that has to reproduce our accent exactly.
	t.Run(termenv.TrueColor.Name(), func(t *testing.T) {
		styles := newStyles(&bytes.Buffer{}, termenv.WithProfile(termenv.TrueColor))
		want := "\x1b[38;2;216;84;10m" + value + "\x1b[0m"
		if got := styles.Accent(value); got != want {
			t.Errorf("Accent() = %q, want %q", got, want)
		}
	})

	// The other color profiles only have to keep the visible text and avoid
	// claiming TrueColor support: the exact approximation is up to termenv.
	for _, profile := range []termenv.Profile{termenv.ANSI256, termenv.ANSI} {
		t.Run(profile.Name(), func(t *testing.T) {
			styles := newStyles(&bytes.Buffer{}, termenv.WithProfile(profile))
			got := styles.Accent(value)

			if !strings.Contains(got, termenv.CSI) {
				t.Errorf("Accent() = %q, want styling for the %s profile", got, profile.Name())
			}
			if strings.Contains(got, "38;2;") {
				t.Errorf("Accent() = %q emitted a TrueColor sequence on the %s profile", got, profile.Name())
			}
			if plain := stripANSI(got); plain != value {
				t.Errorf("visible value = %q, want %q", plain, value)
			}
		})
	}

	t.Run(termenv.Ascii.Name(), func(t *testing.T) {
		styles := newStyles(&bytes.Buffer{}, termenv.WithProfile(termenv.Ascii))
		if got := styles.Accent(value); got != value {
			t.Errorf("Accent() = %q, want plain %q", got, value)
		}
	})
}

func TestTableStylesKeepAlignmentAndPlainText(t *testing.T) {
	clearColorEnv(t)

	rows := [][]tableCell{
		{{"NAME", roleHeader}, {"SOURCE", roleHeader}, {"SELECTED BY", roleHeader}},
		{{"liberica@25.0.4.1", roleDefault}, {"javm", roleDim}, {"25, liberica@25", roleAccent}},
		{{"temurin@25.0.4.1", roleDefault}, {"javm", roleDim}, {"temurin@25", roleAccent}},
		{{"graalvm@25.0.2", roleDefault}, {"javm", roleDim}, {"graalvm@25", roleAccent}},
		{{"liberica@25.0.2", roleDim}, {"javm", roleDim}},
		{{"temurin@24.0.2", roleDefault}, {"javm", roleDim}, {"24, temurin@24", roleAccent}},
		{{"temurin@21.0.12.1", roleDefault}, {"javm", roleDim}, {"21, temurin@21", roleAccent}},
	}

	var plain bytes.Buffer
	if err := writeTable(&plain, newStyles(&plain), rows); err != nil {
		t.Fatalf("write plain table: %v", err)
	}

	// layoutTable is the ground truth for the alignment: type/tabwriter over the
	// plain values, with no styling involved.
	want, err := layoutTable(rows)
	if err != nil {
		t.Fatalf("lay out table: %v", err)
	}
	if plain.String() != want {
		t.Errorf("plain output does not match the unstyled layout:\ngot:\n%s\nwant:\n%s", plain.String(), want)
	}

	var styled bytes.Buffer
	styles := newStyles(&styled, termenv.WithProfile(termenv.TrueColor))
	if err := writeTable(&styled, styles, rows); err != nil {
		t.Fatalf("write styled table: %v", err)
	}

	if !strings.Contains(styled.String(), "\x1b[") {
		t.Fatalf("styled table has no escape sequences:\n%s", styled.String())
	}
	if got := stripANSI(styled.String()); got != want {
		t.Errorf("styling changed the table layout:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestStyleTableResolvesRepeatedValuesLeftToRight(t *testing.T) {
	clearColorEnv(t)

	rows := [][]tableCell{
		{{"NAME", roleHeader}, {"SOURCE", roleHeader}, {"SELECTED BY", roleHeader}},
		{{"b-jdk@17", roleDefault}, {"system", roleDim}, {"b-jdk@17", roleAccent}},
	}

	var styled bytes.Buffer
	styles := newStyles(&styled, termenv.WithProfile(termenv.TrueColor))
	if err := writeTable(&styled, styles, rows); err != nil {
		t.Fatalf("write styled table: %v", err)
	}

	row := ""
	for _, line := range strings.Split(styled.String(), "\n") {
		if strings.Contains(line, styles.Accent("b-jdk@17")) {
			row = line
			break
		}
	}
	if row == "" {
		t.Fatalf("no row with the accented value:\n%q", styled.String())
	}

	// The NAME occurrence stays plain and must come before the SELECTED BY one.
	if strings.Index(row, "b-jdk@17") > strings.Index(row, styles.Accent("b-jdk@17")) {
		t.Errorf("the wrong occurrence was styled:\n%q", row)
	}
	if !strings.Contains(row, styles.Dim("system")) {
		t.Errorf("SOURCE is not dimmed:\n%q", row)
	}
}
