package command

import (
	"fmt"
	"io"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/felipebz/javm/cfg"
	"github.com/felipebz/javm/discovery"
	"github.com/felipebz/javm/semver"
	"github.com/spf13/cobra"
)

func NewLsCommand() *cobra.Command {
	var showDetails bool
	cmd := &cobra.Command{
		Use:   "ls",
		Short: "List installed versions",
		RunE: func(cmd *cobra.Command, args []string) error {
			var rng *semver.Range
			if len(args) > 0 {
				var err error
				rng, err = semver.ParseRange(args[0])
				if err != nil {
					return err
				}
			}

			jdks, err := Ls()
			if err != nil {
				return err
			}

			printInstalledVersions(cmd.OutOrStdout(), jdks, rng, showDetails)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&showDetails, "details", "d", false, "Show detailed information about discovered JDKs")
	return cmd
}

var readDir = os.ReadDir

var lsFunc = func() ([]discovery.JDK, error) {
	manager := discovery.NewManagerWithAllSources(
		discovery.GetDefaultCacheFile(cfg.Dir()),
		discovery.DefaultCacheTTL,
	)

	return manager.DiscoverAll()
}

func Ls() ([]discovery.JDK, error) {
	return lsFunc()
}

func LsBestMatch(selector string) (string, error) {
	jdks, err := Ls()
	if err != nil {
		return "", err
	}
	jdk, err := FindBestMatchJDK(jdks, selector)
	if err != nil {
		return "", err
	}
	return jdk.Identifier, nil
}

func FindBestMatchJDK(jdks []discovery.JDK, selector string) (discovery.JDK, error) {
	rng, err := semver.ParseRange(selector)
	if err != nil {
		return discovery.JDK{}, err
	}

	sort.Slice(jdks, func(i, j int) bool {
		v1, err1 := semver.ParseVersion(jdks[i].Version)
		v2, err2 := semver.ParseVersion(jdks[j].Version)
		if err1 == nil && err2 == nil {
			return v2.LessThan(v1)
		}
		return jdks[i].Version > jdks[j].Version
	})

	for _, jdk := range jdks {
		v, err := semver.ParseVersion(jdk.Identifier)
		if err == nil && rng.Contains(v) {
			return jdk, nil
		}
		// Also try Version field if Identifier didn't work (fallback)
		v2, err2 := semver.ParseVersion(jdk.Version)
		if err2 == nil && rng.Contains(v2) {
			return jdk, nil
		}
	}

	return discovery.JDK{}, fmt.Errorf("%s isn't installed", rng)
}

func printInstalledVersions(w io.Writer, jdks []discovery.JDK, rng *semver.Range, showDetails bool) {
	// Filter by range
	var filtered []discovery.JDK
	for _, jdk := range jdks {
		if rng != nil {
			v, err := semver.ParseVersion(jdk.Identifier)
			if err != nil || !rng.Contains(v) {
				continue
			}
		}
		filtered = append(filtered, jdk)
	}

	// Sort by Source (ASC) then Version (DESC)
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].Source != filtered[j].Source {
			return filtered[i].Source < filtered[j].Source
		}
		v1, err1 := semver.ParseVersion(filtered[i].Version)
		v2, err2 := semver.ParseVersion(filtered[j].Version)
		if err1 == nil && err2 == nil {
			return v2.LessThan(v1)
		}
		return filtered[i].Version > filtered[j].Version
	})

	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
	if showDetails {
		fmt.Fprintln(tw, "SOURCE\tNAME\tVENDOR\tARCHITECTURE\tPATH")
		for _, jdk := range jdks {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
				jdk.Source,
				jdk.Identifier,
				jdk.Vendor,
				jdk.Architecture,
				jdk.Path,
			)
		}
	} else {
		fmt.Fprintln(tw, "NAME\tSOURCE")
		for _, jdk := range jdks {
			fmt.Fprintf(tw, "%s\t%s\n", jdk.Identifier, jdk.Source)
		}
	}
	tw.Flush()
}
