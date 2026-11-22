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
	latest       string
}

func NewLsRemoteCommand(client PackagesClient) *cobra.Command {
	var opts lsRemoteOptions

	defaultDistribution, _ := cfg.EffectiveValue("java.default_distribution")

	cmd := &cobra.Command{
		Use:   "ls-remote",
		Short: "List remote versions available for install",
		Args:  UsageArgs(cobra.MaximumNArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			rangeArg := ""
			if len(args) > 0 {
				rangeArg = args[0]
			}
			normalized := opts
			normalized.os = normalizeOS(opts.os)
			return runLsRemote(cmd.Context(), cmd.OutOrStdout(), client, rangeArg, normalized)
		},
	}
	cmd.Flags().StringVar(&opts.os, "os", runtime.GOOS, "Operating System (macos, linux, windows)")
	cmd.Flags().StringVar(&opts.arch, "arch", runtime.GOARCH, "Architecture (amd64, arm64)")
	cmd.Flags().StringVar(&opts.distribution, "distribution", defaultDistribution, "Java distribution (e.g. temurin, zulu, corretto). Use \"all\" to list all distributions")
	cmd.Flags().StringVar(&opts.latest, "latest", "major",
		"Part of the version to trim to (\"major\", \"minor\" or \"patch\")")
	return cmd
}

func runLsRemote(ctx context.Context, out io.Writer, client PackagesClient, rangeArg string, opts lsRemoteOptions) error {
	var r *javaversion.Range
	var err error
	if rangeArg != "" {
		r, err = javaversion.ParseRange(rangeArg)
		if err != nil {
			return UsageError(err)
		}
	}

	distribution := opts.distribution
	if distribution == "all" {
		distribution = ""
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
