package command

import (
	"context"
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
		Args:  UsageArgs(cobra.MaximumNArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			var rng *semver.Range
			if len(args) > 0 {
				var err error
				rng, err = semver.ParseRange(args[0])
				if err != nil {
					return UsageError(err)
				}
			}

			jdks, err := LsContext(cmd.Context(), false)
			if err != nil {
				return err
			}

			return printInstalledVersions(cmd.OutOrStdout(), jdks, rng, showDetails)
		},
	}
	cmd.Flags().BoolVarP(&showDetails, "details", "d", false, "Show detailed information about discovered JDKs")
	return cmd
}

var readDir = os.ReadDir

var lsFunc = func(ctx context.Context) ([]discovery.JDK, error) {
	manager, err := discovery.NewConfiguredManager(cfg.Dir())
	if err != nil {
		return nil, fmt.Errorf("load discovery configuration: %w", err)
	}
	manager.Warn = func(err error) {
		loggerFromContext(ctx).Warn(err)
	}

	return manager.DiscoverAll()
}

func Ls(managedOnly bool) ([]discovery.JDK, error) {
	return LsContext(context.Background(), managedOnly)
}

func LsContext(ctx context.Context, managedOnly bool) ([]discovery.JDK, error) {
	if managedOnly {
		return discovery.NewJavmSource().Discover()
	}
	return lsFunc(ctx)
}

func LsBestMatch(selector string, managedOnly bool) (string, error) {
	return LsBestMatchContext(context.Background(), selector, managedOnly)
}

func LsBestMatchContext(ctx context.Context, selector string, managedOnly bool) (string, error) {
	jdks, err := LsContext(ctx, managedOnly)
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
		return discovery.JDK{}, UsageError(err)
	}

	sort.Slice(jdks, func(i, j int) bool {
		v1, err1 := semver.ParseVersion(jdks[i].Version)
		v2, err2 := semver.ParseVersion(jdks[j].Version)
		if err1 == nil && err2 == nil {
			return v2.LessThan(v1)
		}
		return jdks[i].Version > jdks[j].Version
	})

	var fallback discovery.JDK
	hasFallback := false

	for _, jdk := range jdks {
		v, err := semver.ParseVersion(jdk.Identifier)
		if err != nil {
			v, err = semver.ParseVersion(jdk.Version)
		}

		if err == nil && rng.Contains(v) {
			if jdk.Source == "javm" {
				return jdk, nil
			}

			if !hasFallback {
				fallback = jdk
				hasFallback = true
			}
		}
	}

	if hasFallback {
		return fallback, nil
	}

	return discovery.JDK{}, NotFoundError(fmt.Errorf("%s isn't installed", rng))
}

func printInstalledVersions(w io.Writer, jdks []discovery.JDK, rng *semver.Range, showDetails bool) error {
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
		if _, err := fmt.Fprintln(tw, "SOURCE\tNAME\tVENDOR\tARCHITECTURE\tPATH"); err != nil {
			return fmt.Errorf("write installed JDK header: %w", err)
		}
		for _, jdk := range filtered {
			if _, err := fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
				jdk.Source,
				jdk.Identifier,
				jdk.Vendor,
				jdk.Architecture,
				jdk.Path,
			); err != nil {
				return fmt.Errorf("write installed JDK: %w", err)
			}
		}
	} else {
		if _, err := fmt.Fprintln(tw, "NAME\tSOURCE"); err != nil {
			return fmt.Errorf("write installed JDK header: %w", err)
		}
		for _, jdk := range filtered {
			if _, err := fmt.Fprintf(tw, "%s\t%s\n", jdk.Identifier, jdk.Source); err != nil {
				return fmt.Errorf("write installed JDK: %w", err)
			}
		}
	}
	if err := tw.Flush(); err != nil {
		return fmt.Errorf("flush installed JDK output: %w", err)
	}
	return nil
}
