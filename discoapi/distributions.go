package discoapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"strings"
)

func (c *Client) GetDistributions() ([]Distribution, error) {
	return c.GetDistributionsContext(context.Background())
}

func (c *Client) GetDistributionsContext(ctx context.Context) ([]Distribution, error) {
	params := url.Values{}
	params.Set("include_versions", "false")
	data, err := c.fetchContext(ctx, "distributions", params)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch distributions: %w", err)
	}

	var response DistributionsResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse distributions: %w", err)
	}

	slices.SortFunc(response.Distributions, func(a, b Distribution) int {
		return strings.Compare(a.APIParameter, b.APIParameter)
	})

	return response.Distributions, nil
}
