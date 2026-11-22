package main

import (
	"io"
	"regexp"
	"strings"
	"testing"
	"unicode"

	"github.com/felipebz/javm/discoapi"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// <name> is required, [name] optional, [args...] repeats; flags shown in the
// Use line take one value.
var useLine = regexp.MustCompile(`^[a-z][a-z-]*( (<[a-z][a-z-]*>|\[[a-z][a-z-]*\]|\[--[a-z-]+ <[a-z-]+>\]|\[--\]|\[[a-z][a-z-]*\.\.\.\]))*$`)

// ls-remote documents its two selector shapes as alternatives.
var useLineExceptions = map[string]bool{
	"ls-remote [<distribution>[@<range>] | <range>]": true,
}

func TestHelpTextFollowsTheConventions(t *testing.T) {
	root := newRootCommand(application{out: io.Discard, err: io.Discard, logger: log.New(), client: discoapi.NewClient()})
	var walk func(*cobra.Command)
	walk = func(cmd *cobra.Command) {
		path := cmd.CommandPath()
		if cmd != root && !useLine.MatchString(cmd.Use) && !useLineExceptions[cmd.Use] {
			t.Errorf("%s: Use %q does not follow <required> [optional] [repeated...]", path, cmd.Use)
		}
		if short := cmd.Short; cmd != root && (short == "" || !unicode.IsUpper([]rune(short)[0]) ||
			strings.HasSuffix(short, ".") || strings.ContainsAny(short, "`\"")) {
			t.Errorf("%s: Short %q must start uppercase, without trailing period, backticks or double quotes", path, short)
		}
		cmd.LocalFlags().VisitAll(func(flag *pflag.Flag) {
			if flag.Hidden || flag.Usage == "" {
				return
			}
			if first := []rune(flag.Usage)[0]; !unicode.IsLower(first) && !unicode.IsDigit(first) || strings.HasSuffix(flag.Usage, ".") {
				t.Errorf("%s --%s: description %q must start lowercase, without trailing period", path, flag.Name, flag.Usage)
			}
		})
		for _, child := range cmd.Commands() {
			walk(child)
		}
	}
	walk(root)
}
