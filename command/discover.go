package command

import (
	"fmt"
	"github.com/felipebz/javm/cfg"
	"github.com/felipebz/javm/discovery"
	"github.com/spf13/cobra"
	"sort"
	"text/tabwriter"
	"time"
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

			if showDetails {
				w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
				fmt.Fprintln(w, "SOURCE\tVERSION\tVENDOR\tIMPLEMENTATION\tARCHITECTURE\tPATH")
				for _, jdk := range jdks {
					fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
						jdk.Source,
						jdk.Version,
						jdk.Vendor,
						jdk.Implementation,
						jdk.Architecture,
						jdk.Path,
					)
				}
				w.Flush()
			} else {
				for _, jdk := range jdks {
					fmt.Fprintf(cmd.OutOrStdout(), "%s@%s\n", jdk.Source, discovery.CleanupVersionString(jdk.Version))
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&showDetails, "details", "d", false, "Show detailed information about discovered JDKs")

	return cmd
}
