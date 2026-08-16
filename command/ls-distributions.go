package command

import (
	"context"
	"fmt"
	"io"

	"github.com/felipebz/javm/discoapi"
	"github.com/spf13/cobra"
)

type DistributionsClient interface {
	GetDistributionsContext(ctx context.Context) ([]discoapi.Distribution, error)
}

func NewLsDistributionsCommand(client DistributionsClient) *cobra.Command {
	return &cobra.Command{
		Use:   "ls-distributions",
		Short: "List all available Java distributions",
		Args:  UsageArgs(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			distributions, err := client.GetDistributionsContext(cmd.Context())
			if err != nil {
				return NetworkError(err)
			}
			return printDistributions(cmd.OutOrStdout(), distributions)
		},
	}
}

func printDistributions(w io.Writer, distributions []discoapi.Distribution) error {
	if _, err := fmt.Fprintf(w, "%-20s %s\n", "Identifier", "Name"); err != nil {
		return fmt.Errorf("write distribution header: %w", err)
	}
	for _, dist := range distributions {
		if _, err := fmt.Fprintf(w, "%-20s %s\n", dist.APIParameter, dist.Name); err != nil {
			return fmt.Errorf("write distribution: %w", err)
		}
	}
	return nil
}
