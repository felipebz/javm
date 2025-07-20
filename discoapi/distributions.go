package discoapi

import (
	"encoding/json"
	"net/url"
	"slices"
	"strings"
)

func (c *Client) GetDistributions() ([]Distribution, error) {
	params := url.Values{}
	params.Set("include_versions", "false")
	data, err := c.fetch("distributions", params)
	if err != nil {
		return nil, err
	}

	var response DistributionsResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	slices.SortFunc(response.Distributions, func(a, b Distribution) int {
		return strings.Compare(a.APIParameter, b.APIParameter)
	})

	return response.Distributions, nil
}
