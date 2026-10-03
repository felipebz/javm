package command

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felipebz/javm/discoapi"
	"github.com/felipebz/javm/discovery"
)

// hostileTerminalPayload is the escape and OSC-52 clipboard sequence class that
// a compromised package index or JDK archive can place in JDK metadata.
const hostileTerminalPayload = "Eclipse Adoptium\x1b[31mINJECTED\x1b]52;c;cHduZWQ=\x07"

// containsTerminalEscape reports whether s carries a control character that is
// not a row separator, which is what must never reach the terminal.
func containsTerminalEscape(s string) bool {
	return strings.ContainsFunc(s, func(r rune) bool {
		return r != '\n' && isTerminalControl(r)
	})
}

func TestTableRendersControlCharactersInert(t *testing.T) {
	clearColorEnv(t)
	const inert = "Eclipse Adoptium[31mINJECTED]52;c;cHduZWQ="

	rows := [][]tableCell{
		{{"VENDOR", roleHeader}, {"PATH", roleHeader}},
		{{hostileTerminalPayload, roleDefault}, {"/jdks/temurin@21.0.1", roleDefault}},
		{{"BellSoft", roleDim}, {"/jdks/liberica@21.0.1", roleDefault}},
	}

	// The column width has to come from the sanitized text, so the padded layout
	// is the expected one even though the raw value is longer.
	want := "VENDOR" + strings.Repeat(" ", len(inert)-len("VENDOR")+3) + "PATH\n" +
		inert + strings.Repeat(" ", 3) + "/jdks/temurin@21.0.1\n" +
		"BellSoft" + strings.Repeat(" ", len(inert)-len("BellSoft")+3) + "/jdks/liberica@21.0.1\n"

	var plain bytes.Buffer
	if err := writeTable(&plain, newStyles(&plain), rows); err != nil {
		t.Fatal(err)
	}
	if got := plain.String(); got != want {
		t.Fatalf("plain table =\n%q\nwant\n%q", got, want)
	}

	t.Setenv("CLICOLOR_FORCE", "1")
	var styled bytes.Buffer
	if err := writeTable(&styled, newStyles(&styled), rows); err != nil {
		t.Fatal(err)
	}
	got := styled.String()
	if strings.Contains(got, hostileTerminalPayload) {
		t.Fatalf("styled table emitted the raw payload: %q", got)
	}
	if plain := stripANSI(got); plain != want {
		t.Fatalf("styled table visible text =\n%q\nwant\n%q", plain, want)
	}
	if !strings.Contains(got, inert) {
		t.Fatalf("styled table did not render the payload as inert text: %q", got)
	}
}

func TestLsDetailsRendersHostileMetadataInert(t *testing.T) {
	cleanup := setupMockLs()
	defer cleanup()
	clearColorEnv(t)

	mockLsResult = []discovery.JDK{{
		Identifier:   "temurin@21.0.1",
		Version:      "21.0.1",
		Source:       "javm",
		Vendor:       hostileTerminalPayload,
		Architecture: "x64\x1b[2J",
		Path:         "/jdks/\x1b]0;pwned\x07temurin@21.0.1",
	}}

	cmd := NewLsCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--details"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	got := out.String()
	if containsTerminalEscape(got) {
		t.Fatalf("ls --details emitted control characters: %q", got)
	}
	for _, want := range []string{
		"Eclipse Adoptium[31mINJECTED]52;c;cHduZWQ=",
		"x64[2J",
		"/jdks/]0;pwnedtemurin@21.0.1",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("ls --details output %q does not contain %q", got, want)
		}
	}
}

func TestWhichRendersHostileJDKPathInert(t *testing.T) {
	cleanup := setupMockLs()
	defer cleanup()
	clearColorEnv(t)

	mockLsResult = []discovery.JDK{{
		Identifier:   "temurin@21.0.1",
		Version:      "21.0.1",
		Source:       "javm",
		Vendor:       "Eclipse Adoptium",
		Architecture: "x64",
		Path:         "/jdks/\x1b]52;c;cHduZWQ=\x07temurin@21.0.1",
	}}

	cmd := NewWhichCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"temurin@21"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	got := out.String()
	if containsTerminalEscape(got) {
		t.Fatalf("which emitted control characters: %q", got)
	}
	if want := "/jdks/]52;c;cHduZWQ=temurin@21.0.1\n"; got != want {
		t.Fatalf("which output = %q, want %q", got, want)
	}
}

func TestCurrentRendersHostileJDKNameInert(t *testing.T) {
	clearColorEnv(t)
	home := t.TempDir()
	t.Setenv("JAVM_HOME", home)

	original := lookPath
	lookPath = func(string) (string, error) {
		return filepath.Join(home, "jdk", "temurin@21.0.1\x1b[31mINJECTED", "bin", "java"), nil
	}
	defer func() { lookPath = original }()

	cmd := NewCurrentCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	got := out.String()
	if containsTerminalEscape(got) {
		t.Fatalf("current emitted control characters: %q", got)
	}
	if want := "temurin@21.0.1[31mINJECTED\n"; got != want {
		t.Fatalf("current output = %q, want %q", got, want)
	}
}

func TestLsRemoteRendersHostileMetadataInert(t *testing.T) {
	clearColorEnv(t)
	client := &mockPackagesClient{Pkgs: []discoapi.Package{{
		JavaVersion:         "21.0.1",
		Distribution:        "temurin",
		DistributionVersion: "21.0.1\x1b]52;c;cHduZWQ=\x07",
	}}}

	cmd := NewLsRemoteCommand(client)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--os=linux", "--arch=amd64"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	got := out.String()
	if containsTerminalEscape(got) {
		t.Fatalf("ls-remote emitted control characters: %q", got)
	}
	if want := "temurin 21.0.1]52;c;cHduZWQ="; !strings.Contains(got, want) {
		t.Fatalf("ls-remote output %q does not contain %q", got, want)
	}
}

func TestLsDistributionsRendersHostileFieldsInert(t *testing.T) {
	var out bytes.Buffer
	distributions := []discoapi.Distribution{
		{APIParameter: "zulu\x1b[31m", Name: "Azul\x1b]0;pwned\x07"},
	}
	if err := printDistributions(&out, distributions); err != nil {
		t.Fatal(err)
	}

	got := out.String()
	if containsTerminalEscape(got) {
		t.Fatalf("ls-distributions emitted control characters: %q", got)
	}
	if want := "zulu[31m"; !strings.Contains(got, want) {
		t.Fatalf("ls-distributions output %q does not contain %q", got, want)
	}
	if want := "Azul]0;pwned"; !strings.Contains(got, want) {
		t.Fatalf("ls-distributions output %q does not contain %q", got, want)
	}
}
