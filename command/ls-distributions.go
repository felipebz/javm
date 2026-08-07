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
		RunE: func(cmd *cobra.Command, args []string) error {
			distributions, err := client.GetDistributionsContext(cmd.Context())
			if err != nil {
				return err
			}
			printDistributions(cmd.OutOrStdout(), distributions)
			return nil
		},
	}
}

func printDistributions(w io.Writer, distributions []discoapi.Distribution) {
	fmt.Fprintf(w, "%-20s %s\n", "Identifier", "Name")
	for _, dist := range distributions {
		fmt.Fprintf(w, "%-20s %s\n", dist.APIParameter, dist.Name)
	}
}
