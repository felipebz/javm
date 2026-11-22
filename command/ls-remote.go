package command

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"strings"

	"github.com/felipebz/javm/cfg"
	"github.com/felipebz/javm/javaversion"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"
)

type lsRemoteOptions struct {
	os           string
	arch         string
	distribution string
	// distributionFlag is set when --distribution was given explicitly.
	distributionFlag bool
	latest           string
}

func NewLsRemoteCommand(client PackagesClient) *cobra.Command {
	var opts lsRemoteOptions

	defaultDistribution, _ := cfg.EffectiveValue("java.default_distribution")

	cmd := &cobra.Command{
		Use:   "ls-remote [<distribution>[@<range>] | <range>]",
		Short: "List JDK versions available for install",
		Long: "List the versions available for install. A distribution name lists all its versions, <distribution>@<range>\n" +
			"filters them, and a range alone uses java.default_distribution. 'all' lists every distribution.",
		Args: UsageArgs(cobra.MaximumNArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			selector := ""
			if len(args) > 0 {
				selector = args[0]
			}
			normalized := opts
			normalized.os = normalizeOS(opts.os)
			normalized.distributionFlag = cmd.Flags().Changed("distribution")
			return runLsRemote(cmd.Context(), cmd.OutOrStdout(), client, selector, normalized)
		},
		Example: "  javm ls-remote\n" +
			"  javm ls-remote 21\n" +
			"  javm ls-remote zulu\n" +
			"  javm ls-remote zulu@25\n" +
			"  javm ls-remote all@21",
	}
	cmd.Flags().StringVar(&opts.os, "os", runtime.GOOS, "operating system (macos, linux, windows)")
	cmd.Flags().StringVar(&opts.arch, "arch", runtime.GOARCH, "architecture (amd64, arm64)")
	cmd.Flags().StringVar(&opts.distribution, "distribution", defaultDistribution, "deprecated: name the distribution in the selector instead (javm ls-remote zulu@25)")
	cmd.Flags().StringVar(&opts.latest, "latest", "major",
		"part of the version to trim to (major, minor or patch)")
	return cmd
}

// parseLsRemoteSelector splits the selector into the distribution it names
// ("" when none) and the range filter. A bare word that is not a range is a
// distribution when "<word>@" parses, so the same names install accepts work.
func parseLsRemoteSelector(selector string) (string, *javaversion.Range, error) {
	if selector == "" {
		return "", nil, nil
	}
	r, err := javaversion.ParseRange(selector)
	if err != nil && !strings.Contains(selector, "@") {
		if bare, bareErr := javaversion.ParseRange(selector + "@"); bareErr == nil {
			r, err = bare, nil
		}
	}
	if err != nil {
		return "", nil, UsageError(err)
	}
	distribution := r.Qualifier
	if distribution == "*" {
		distribution = "all"
	}
	return distribution, r, nil
}

func runLsRemote(ctx context.Context, out io.Writer, client PackagesClient, selector string, opts lsRemoteOptions) error {
	distribution, r, err := parseLsRemoteSelector(selector)
	if err != nil {
		return err
	}
	if opts.distributionFlag {
		if err := warnDistributionFlag(ctx, selector, distribution, opts.distribution); err != nil {
			return err
		}
	}
	if distribution == "" {
		distribution = opts.distribution
	}
	if distribution == "all" {
		distribution = ""
		if r != nil {
			r.Qualifier = ""
		}
	}
	packageIndex, err := makePackageIndex(ctx, client, opts.os, opts.arch, distribution)
	if err != nil {
		return err
	}
	if packageIndex.knownEmpty {
		if _, err := fmt.Fprintf(RuntimeFromContext(ctx).Err, "no versions of %q found for %s/%s\n",
			distribution, sanitizeTerminalText(opts.os), sanitizeTerminalText(opts.arch)); err != nil {
			return fmt.Errorf("write remote JDK status: %w", err)
		}
	}

	trimToValue := parseTrimTo(opts.latest)
	if opts.latest != "" && trimToValue < 0 {
		return UsageError(fmt.Errorf("invalid value for --latest %q: want major, minor or patch", opts.latest))
	}
	vs := packageIndex.Sorted
	if opts.latest != "" {
		vs = packageIndex.trimTo(vs, trimToValue)
	}

	return printVersions(out, vs, packageIndex, r, trimToValue)
}

// warnDistributionFlag rejects a --distribution that conflicts with the
// selector and otherwise warns that the flag is deprecated, suggesting the
// selector form.
func warnDistributionFlag(ctx context.Context, selector, distribution, flag string) error {
	if distribution != "" && distribution != flag {
		return UsageError(fmt.Errorf("--distribution %q conflicts with the distribution %q in the selector", flag, distribution))
	}
	newForm := flag
	if distribution == "" && selector != "" {
		newForm += "@" + selector
	} else if selector != "" {
		newForm = selector
	}
	hint := "use: javm ls-remote " + shellWord(sanitizeTerminalText(newForm))
	if strings.ContainsAny(newForm, "\"$`\\") && strings.Contains(newForm, "'") {
		hint = "name the distribution in the selector"
	}
	warnOnce(ctx, fmt.Errorf("--distribution is deprecated; %s", hint))
	return nil
}

// shellWord returns s ready to paste into a POSIX shell, PowerShell or cmd: as
// is when it only has characters none of them treats specially, otherwise
// quoted. "," and "^" are quoted too: PowerShell splits on "," and cmd
// consumes "^".
func shellWord(s string) string {
	safe := s != "" && !strings.ContainsFunc(s, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("@._-+~", r))
	})
	switch {
	case safe:
		return s
	case !strings.ContainsAny(s, "\"$`\\"):
		return `"` + s + `"`
	default:
		return "'" + s + "'"
	}
}

func printVersions(out io.Writer, versions []*javaversion.Version, packageIndex *packageIndex, r *javaversion.Range, value javaversion.VersionPart) error {
	rows := [][]tableCell{{
		{"Identifier", roleHeader},
		{"Full Version", roleHeader},
		{"Distribution", roleHeader},
		{"Distribution Version", roleHeader},
	}}
	for _, v := range versions {
		if r != nil && !r.Contains(v) {
			continue
		}
		pkg := packageIndex.ByVersion[v]
		rows = append(rows, []tableCell{
			{sanitizeTerminalText(v.TrimTo(value)), roleDefault},
			{sanitizeTerminalText(pkg.JavaVersion), roleDefault},
			{sanitizeTerminalText(pkg.Distribution), roleDefault},
			{sanitizeTerminalText(pkg.DistributionVersion), roleDefault},
		})
	}
	// Without a matching version nothing is printed, not even the header.
	if len(rows) == 1 {
		return nil
	}
	if err := writeTable(out, newStyles(out, termenv.WithProfile(termenv.Ascii)), rows); err != nil {
		return fmt.Errorf("write remote JDKs: %w", err)
	}
	return nil
}

func normalizeOS(os string) string {
	switch strings.ToLower(os) {
	case "macos", "osx", "mac", "macosx":
		return "darwin"
	case "win":
		return "windows"
	default:
		return os
	}
}
