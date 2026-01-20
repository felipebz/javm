package discoapi

import (
	"encoding/json"
	"fmt"
	"net/url"

	log "github.com/sirupsen/logrus"
)

func (c *Client) GetPackages(os, arch, distribution, version string) ([]Package, error) {
	archFilter := map[string]string{
		"amd64": "amd64,x64",
		"arm64": "arm64,aarch64",
	}[arch]

	params := url.Values{}
	if os != "" {
		params.Set("operating_system", os)
	}
	if arch != "" {
		params.Set("architecture", archFilter)
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

		libc := "glibc"
		if os == "linux" && isMuslLibc() {
			libc = "musl"
		} else if os == "darwin" {
			libc = "libc"
		}

		log.Debugf("OS is %s, libc is %s", os, libc)
		params.Set("lib_c_type", libc)
	}

	params.Set("package_type", "jdk")
	params.Set("release_status", "ga")

	log.Debugf("fetching packages with params: %s", params.Encode())
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

func (c *Client) GetPackageInfo(id string) (*PackageInfo, error) {
	data, err := c.fetch("ids/"+id, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch package info: %w", err)
	}

	var resp PackageInfoResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse package info: %w", err)
	}

	return &resp.PackageInfo[0], nil
}
