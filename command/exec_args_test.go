package command

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/pflag"
)

func TestParseExecArgsOptionGrammar(t *testing.T) {
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, ".java-version"), []byte("21\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(workspace)

	tests := []struct {
		args     []string
		selector string
		command  []string
		quiet    bool
		debug    bool
	}{
		{args: []string{"--jdk", "17", "cmd"}, selector: "17", command: []string{"cmd"}},
		{args: []string{"--jdk=17", "cmd"}, selector: "17", command: []string{"cmd"}},
		{args: []string{"--", "-cmd"}, selector: "21", command: []string{"-cmd"}},
		{args: []string{"--jdk", "17", "--", "cmd", "--flag"}, selector: "17", command: []string{"cmd", "--flag"}},
		{args: []string{"cmd", "--help"}, selector: "21", command: []string{"cmd", "--help"}},
		{args: []string{"cmd", "--", "x"}, selector: "21", command: []string{"cmd", "--", "x"}},
		{args: []string{"--quiet", "--jdk", "17", "cmd"}, selector: "17", command: []string{"cmd"}, quiet: true},
		{args: []string{"--debug", "cmd", "--quiet"}, selector: "21", command: []string{"cmd", "--quiet"}, debug: true},
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			got, err := parseExecArgs(tt.args)
			if err != nil {
				t.Fatal(err)
			}
			if got.selector != tt.selector || !slices.Equal(got.command, tt.command) || got.quiet != tt.quiet || got.debug != tt.debug {
				t.Fatalf("parseExecArgs(%q) = %+v, want selector %q, command %q, quiet %v, debug %v",
					tt.args, got, tt.selector, tt.command, tt.quiet, tt.debug)
			}
		})
	}
}

func TestParseExecArgsHelpAndUsageErrors(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"-h"}, {"--jdk", "17", "--help"}, {"--jdk=17", "-h", "cmd"}} {
		if _, err := parseExecArgs(args); !errors.Is(err, pflag.ErrHelp) {
			t.Errorf("parseExecArgs(%q) = %v, want the help", args, err)
		}
	}
	tests := []struct {
		args []string
		want string
	}{
		{args: []string{"--bogus", "cmd"}, want: `unknown exec option "--bogus"`},
		{args: []string{"-x", "cmd"}, want: "--"},
		{args: []string{"--jdk="}, want: "--jdk requires a selector"},
		{args: []string{"--jdk"}, want: "--jdk requires a selector"},
		{args: nil, want: "no command specified"},
		{args: []string{"--jdk", "17"}, want: "no command specified"},
		{args: []string{"--jdk", "17", "--"}, want: "no command specified"},
	}
	for _, tt := range tests {
		if _, err := parseExecArgs(tt.args); !errors.Is(err, ErrUsage) || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("parseExecArgs(%q) = %v, want a usage error containing %q", tt.args, err, tt.want)
		}
	}
}
