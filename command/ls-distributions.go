package command

import (
	"fmt"
	"io"

	"github.com/felipebz/javm/discoapi"
)

type DistributionsClient interface {
	GetDistributions() ([]discoapi.Distribution, error)
}

func LsDistributions(client DistributionsClient) ([]discoapi.Distribution, error) {
	return client.GetDistributions()
}

func PrintDistributions(w io.Writer, distributions []discoapi.Distribution) {
	fmt.Fprintf(w, "%-20s %s\n", "Identifier", "Name")
	for _, dist := range distributions {
		fmt.Fprintf(w, "%-20s %s\n", dist.APIParameter, dist.Name)
	}
}
