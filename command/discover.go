package command

import (
	"fmt"

	"github.com/felipebz/javm/cfg"
	"github.com/felipebz/javm/discovery"
	"github.com/spf13/cobra"
)

type discoverRunner interface {
	DiscoverAll() ([]discovery.JDK, error)
}

var newManagerWithAllSources = func(configDir string, forceRefresh bool, warn func(error)) (discoverRunner, error) {
	manager, err := discovery.NewConfiguredManager(configDir)
	if err != nil {
		return nil, err
	}
	manager.IgnoreCache = forceRefresh
	manager.Warn = warn
	return manager, nil
}

func NewDiscoverCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "discover",
		Short: "Manage JDK discovery",
		Long:  "Discover JDK installations on the system",
		Args:  UsageArgs(cobra.NoArgs),
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
		Args:  UsageArgs(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			manager, err := newManagerWithAllSources(
				cfg.Dir(),
				true,
				func(err error) { loggerFromContext(cmd.Context()).Warn(err) },
			)
			if err != nil {
				return fmt.Errorf("failed to load discovery configuration: %w", err)
			}

			_, err = manager.DiscoverAll()
			if err != nil {
				return fmt.Errorf("failed to refresh discovery cache: %w", err)
			}
			if _, err := fmt.Fprintln(cmd.OutOrStdout(), "Discovery cache refreshed successfully"); err != nil {
				return fmt.Errorf("write discovery result: %w", err)
			}
			return nil
		},
	}
}
