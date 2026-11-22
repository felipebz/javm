package command

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/felipebz/javm/discoapi"
)

type updateDistributionsClient struct {
	updateClient
}

func (c *updateDistributionsClient) GetPackagesContext(_ context.Context, _, _, distribution, _ string) ([]discoapi.Package, error) {
	var packages []discoapi.Package
	for _, pkg := range c.packages {
		if pkg.Distribution == distribution {
			packages = append(packages, pkg)
		}
	}
	return packages, nil
}

func (c *updateDistributionsClient) GetDistributionsContext(context.Context) ([]discoapi.Distribution, error) {
	return []discoapi.Distribution{{Name: "Temurin", APIParameter: "temurin"}}, nil
}

func goneDistributionFixture(t *testing.T) *updateDistributionsClient {
	t.Helper()
	client := &updateDistributionsClient{updateClient: *temurinUpdateFixture(t)}
	writeManagedJDK(t, "gone@17.0.12", "17.0.12+7", "")
	return client
}

func TestUpdateAllSkipsDistributionNoLongerOffered(t *testing.T) {
	client := goneDistributionFixture(t)
	ctx, logs := contextWithLogBuffer(context.Background())

	if err := runUpdate(ctx, client, &bytes.Buffer{}, "", updateOptions{}); err != nil {
		t.Fatalf("runUpdate() error = %v, want the other lines updated and no error", err)
	}
	assertExists(t, "temurin@25.0.5", true)
	assertExists(t, "gone@17.0.12", true)
	if got := countLines(logs, `distribution \"gone\" is not offered by the Disco API anymore`); got != 1 {
		t.Fatalf("logs = %q, want one warning explaining the skipped line", logs.String())
	}
}

func TestUpdateExplicitUnknownDistributionIsUsageError(t *testing.T) {
	client := goneDistributionFixture(t)
	err := runUpdate(context.Background(), client, &bytes.Buffer{}, "gone@17", updateOptions{})
	if !errors.Is(err, ErrUsage) || !strings.Contains(err.Error(), `unknown distribution "gone"`) || !strings.Contains(err.Error(), "javm ls-distributions") {
		t.Fatalf("update gone@17 error = %v, want a usage error with the hint", err)
	}
}
