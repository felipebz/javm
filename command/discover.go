package command

import (
	"fmt"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/felipebz/javm/cfg"
	"github.com/felipebz/javm/discovery"
	"github.com/spf13/cobra"
)

type discoverRunner interface {
	DiscoverAll() ([]discovery.JDK, error)
}

var newManagerWithAllSources = func(cacheFile string, cacheTTL time.Duration) discoverRunner {
	return discovery.NewManagerWithAllSources(cacheFile, cacheTTL)
}

func NewDiscoverCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "discover",
		Short: "Manage JDK discovery",
		Long:  "Discover JDK installations on the system",
	}

	cmd.AddCommand(
		newDiscoverRefreshCommand(),
		newDiscoverListCommand(),
	)

	return cmd
}

func newDiscoverRefreshCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "refresh",
		Short: "Refresh the discovery cache",
		Long:  "Force a refresh of the JDK discovery cache",
		RunE: func(cmd *cobra.Command, args []string) error {
			manager := newManagerWithAllSources(
				discovery.GetDefaultCacheFile(cfg.Dir()),
				0, // Set TTL to 0 to force refresh
			)

			_, err := manager.DiscoverAll()
			if err != nil {
				return fmt.Errorf("failed to refresh discovery cache: %w", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Discovery cache refreshed successfully")
			return nil
		},
	}
}

func newDiscoverListCommand() *cobra.Command {
	var showDetails bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List discovered JDKs",
		Long:  "List JDK installations discovered on the system",
		RunE: func(cmd *cobra.Command, args []string) error {
			manager := newManagerWithAllSources(
				discovery.GetDefaultCacheFile(cfg.Dir()),
				discovery.DefaultCacheTTL,
			)

			jdks, err := manager.DiscoverAll()
			if err != nil {
				return fmt.Errorf("failed to discover JDKs: %w", err)
			}

			sort.Slice(jdks, func(i, j int) bool {
				if jdks[i].Source != jdks[j].Source {
					return jdks[i].Source < jdks[j].Source
				}
				return jdks[i].Version < jdks[j].Version
			})

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			if showDetails {
				fmt.Fprintln(w, "SOURCE\tNAME\tVENDOR\tARCHITECTURE\tPATH")
				for _, jdk := range jdks {
					fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
						jdk.Source,
						jdk.Identifier,
						jdk.Vendor,
						jdk.Architecture,
						jdk.Path,
					)
				}
			} else {
				fmt.Fprintln(w, "NAME\tSOURCE")
				for _, jdk := range jdks {
					fmt.Fprintf(w, "%s\t%s\n", jdk.Identifier, jdk.Source)
				}
			}
			w.Flush()

			return nil
		},
	}

	cmd.Flags().BoolVarP(&showDetails, "details", "d", false, "Show detailed information about discovered JDKs")

	return cmd
}
