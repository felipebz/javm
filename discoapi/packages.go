package discoapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"os/exec"
	"runtime"

	log "github.com/sirupsen/logrus"
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
		// Default to glibc for non-Windows, override to musl when detected at runtime on Linux
		libc := "glibc"
		if runtime.GOOS == "linux" && isMuslLibc() {
			libc = "musl"
		}
		log.Debugf("OS is %s, libc is %s", runtime.GOOS, libc)
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

// isMuslLibc attempts to detect musl-based Linux by inspecting `ldd --version` output.
// It best-effort returns false if the command is unavailable or any error occurs.
func isMuslLibc() bool {
	cmd := exec.Command("ldd", "--version")
	out, _ := cmd.CombinedOutput()
	return bytes.Contains(out, []byte("musl"))
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
