package command

import (
	"fmt"
	"io"
	"runtime"

	"github.com/felipebz/javm/semver"
	"github.com/spf13/cobra"
)

func NewLsRemoteCommand(client PackagesClient) *cobra.Command {
	var trimTo string
	var osFlag string
	var archFlag string
	var distributionFlag string

	cmd := &cobra.Command{
		Use:   "ls-remote",
		Short: "List remote versions available for install",
		RunE: func(cmd *cobra.Command, args []string) error {
			rangeArg := ""
			if len(args) > 0 {
				rangeArg = args[0]
			}
			return runLsRemote(
				cmd.OutOrStdout(),
				client,
				osFlag,
				archFlag,
				distributionFlag,
				trimTo,
				rangeArg,
			)
		},
	}
	cmd.Flags().StringVar(&osFlag, "os", runtime.GOOS, "Operating System (macos, linux, windows)")
	cmd.Flags().StringVar(&archFlag, "arch", runtime.GOARCH, "Architecture (amd64, arm64)")
	cmd.Flags().StringVar(&distributionFlag, "distribution", "temurin", "Java distribution (e.g. temurin, zulu, corretto). Use \"all\" to list all distributions")
	cmd.Flags().StringVar(&trimTo, "latest", "major",
		"Part of the version to trim to (\"major\", \"minor\" or \"patch\")")
	return cmd
}

func runLsRemote(
	out io.Writer,
	client PackagesClient,
	osFlag, archFlag, distributionFlag, trimTo, rangeArg string,
) error {
	var r *semver.Range
	var err error
	if rangeArg != "" {
		r, err = semver.ParseRange(rangeArg)
		if err != nil {
			return err
		}
	}

	if distributionFlag == "all" {
		distributionFlag = ""
	}
	packageIndex, err := makePackageIndex(client, osFlag, archFlag, distributionFlag)
	if err != nil {
		return err
	}

	trimToValue := parseTrimTo(trimTo)
	vs := packageIndex.Sorted
	if trimTo != "" {
		vs = semver.VersionSlice(vs).TrimTo(trimToValue)
	}

	printVersions(out, vs, packageIndex, r, trimToValue)
	return nil
}

func printVersions(out io.Writer, versions []*semver.Version, packageIndex *packageIndex, r *semver.Range, value semver.VersionPart) {
	headerPrinted := false
	for _, v := range versions {
		if r != nil && !r.Contains(v) {
			continue
		}
		pkg := packageIndex.ByVersion[v]

		if !headerPrinted {
			fmt.Fprintf(out, "%-20s %-15s %s\n", "Identifier", "Full Version", "Distribution Version")
			headerPrinted = true
		}

		fmt.Fprintf(out, "%-20s %-15s %s %s\n", v.TrimTo(value), pkg.JavaVersion, pkg.Distribution, pkg.DistributionVersion)
	}
}
