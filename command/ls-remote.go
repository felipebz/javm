package command

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"strings"

	"github.com/felipebz/javm/cfg"
	"github.com/felipebz/javm/semver"
	"github.com/spf13/cobra"
)

func NewLsRemoteCommand(client PackagesClient) *cobra.Command {
	var trimTo string
	var osFlag string
	var archFlag string
	var distributionFlag string

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

			normalizedOS := normalizeOS(osFlag)
			return runLsRemote(
				cmd.Context(),
				cmd.OutOrStdout(),
				client,
				normalizedOS,
				archFlag,
				distributionFlag,
				trimTo,
				rangeArg,
			)
		},
	}
	cmd.Flags().StringVar(&osFlag, "os", runtime.GOOS, "Operating System (macos, linux, windows)")
	cmd.Flags().StringVar(&archFlag, "arch", runtime.GOARCH, "Architecture (amd64, arm64)")
	cmd.Flags().StringVar(&distributionFlag, "distribution", defaultDistribution, "Java distribution (e.g. temurin, zulu, corretto). Use \"all\" to list all distributions")
	cmd.Flags().StringVar(&trimTo, "latest", "major",
		"Part of the version to trim to (\"major\", \"minor\" or \"patch\")")
	return cmd
}

func runLsRemote(
	ctx context.Context,
	out io.Writer,
	client PackagesClient,
	osFlag, archFlag, distributionFlag, trimTo, rangeArg string,
) error {
	var r *semver.Range
	var err error
	if rangeArg != "" {
		r, err = semver.ParseRange(rangeArg)
		if err != nil {
			return UsageError(err)
		}
	}

	if distributionFlag == "all" {
		distributionFlag = ""
	}
	packageIndex, err := makePackageIndex(ctx, client, osFlag, archFlag, distributionFlag)
	if err != nil {
		return err
	}

	trimToValue := parseTrimTo(trimTo)
	if trimTo != "" && trimToValue < 0 {
		return UsageError(fmt.Errorf("invalid value for --latest %q: want major, minor or patch", trimTo))
	}
	vs := packageIndex.Sorted
	if trimTo != "" {
		vs = semver.VersionSlice(vs).TrimTo(trimToValue)
	}

	return printVersions(out, vs, packageIndex, r, trimToValue)
}

func printVersions(out io.Writer, versions []*semver.Version, packageIndex *packageIndex, r *semver.Range, value semver.VersionPart) error {
	headerPrinted := false
	for _, v := range versions {
		if r != nil && !r.Contains(v) {
			continue
		}
		pkg := packageIndex.ByVersion[v]

		if !headerPrinted {
			if _, err := fmt.Fprintf(out, "%-20s %-15s %s\n", "Identifier", "Full Version", "Distribution Version"); err != nil {
				return fmt.Errorf("write remote JDK header: %w", err)
			}
			headerPrinted = true
		}

		if _, err := fmt.Fprintf(out, "%-20s %-15s %s %s\n", v.TrimTo(value), pkg.JavaVersion, pkg.Distribution, pkg.DistributionVersion); err != nil {
			return fmt.Errorf("write remote JDK: %w", err)
		}
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
