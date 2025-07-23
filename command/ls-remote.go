package command

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io"
	"runtime"
	"sort"
	"strings"

	"github.com/felipebz/javm/discoapi"
	"github.com/felipebz/javm/semver"
)

type PackagesClient interface {
	GetPackages(os, arch, distribution, version string) ([]discoapi.Package, error)
}

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
	cmd.Flags().StringVar(&distributionFlag, "distribution", "", "Java distribution (e.g. temurin, zulu, corretto). Default is 'temurin'")
	cmd.Flags().StringVar(&trimTo, "latest", "",
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

	// Default distribution is "temurin" unless --distribution is set
	filterDistribution := distributionFlag
	if filterDistribution == "" {
		filterDistribution = "temurin"
	}

	packages, err := client.GetPackages(osFlag, archFlag, filterDistribution, "")
	if err != nil {
		return err
	}

	// Build map for version lookup
	releaseMap := make(map[*semver.Version]discoapi.Package)
	for _, pkg := range packages {
		v, err := semver.ParseVersion(pkg.JavaVersion)
		if err != nil {
			continue
		}
		releaseMap[v] = pkg
	}

	vs := make([]*semver.Version, 0, len(releaseMap))
	for v := range releaseMap {
		vs = append(vs, v)
	}
	sort.Sort(semver.VersionSlice(vs))

	if trimTo != "" {
		vs = semver.VersionSlice(vs).TrimTo(parseTrimTo(trimTo))
	}

	printVersions(out, vs, releaseMap, r)
	return nil
}

func printVersions(
	out io.Writer,
	versions []*semver.Version,
	releaseMap map[*semver.Version]discoapi.Package,
	r *semver.Range,
) {
	headerPrinted := false
	for _, v := range versions {
		if r != nil && !r.Contains(v) {
			continue
		}
		pkg := releaseMap[v]

		shortVersion := fmt.Sprintf("%s@%s", pkg.Distribution, stripBuildSuffix(pkg.JavaVersion))

		if !headerPrinted {
			fmt.Fprintf(out, "%-20s %-15s %s\n", "Identifier", "Full Version", "Distribution Version")
			headerPrinted = true
		}

		fmt.Fprintf(out, "%-20s %-15s %s %s\n", shortVersion, pkg.JavaVersion, pkg.Distribution, pkg.DistributionVersion)
	}
}

func stripBuildSuffix(javaVersion string) string {
	if idx := strings.Index(javaVersion, "+"); idx != -1 {
		return javaVersion[:idx]
	}
	return javaVersion
}

func parseTrimTo(value string) semver.VersionPart {
	switch strings.ToLower(value) {
	case "major":
		return semver.VPMajor
	case "minor":
		return semver.VPMinor
	case "patch":
		return semver.VPPatch
	default:
		log.Fatal("Unexpected value of --latest (must be either \"major\", \"minor\" or \"patch\")")
		return -1
	}
}
