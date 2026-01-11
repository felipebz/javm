package command

import (
	"fmt"
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
