package discoapi

import (
	"encoding/json"
	"fmt"
	"net/url"
)

func (c *Client) GetPackages(os, arch, distribution, version string) ([]Package, error) {
	params := url.Values{}
	if os != "" {
		params.Set("operating_system", os)
	}
	if arch != "" {
		params.Set("architecture", arch)
	}
	if distribution != "" {
		params.Set("distribution", distribution)
	}
	if version != "" {
		params.Set("version", version)
	}

	if os == "windows" {
		params.Set("archive_type", "zip")
		params.Set("lib_c_type", "c_std_lib")
	} else {
		params.Set("archive_type", "tar.gz")
		params.Set("lib_c_type", "glibc") // TODO support musl based distros
	}

	params.Set("package_type", "jdk")
	params.Set("release_status", "ga")

	data, err := c.fetch("packages", params)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch packages: %w", err)
	}

	var resp PackagesResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse packages: %w", err)
	}

	return resp.Packages, nil
}
