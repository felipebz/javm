package discoapi

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

// Sample JSON response similar to what DiscoAPI would return
const mockPackagesResponse = `{
  "result": [
    {
      "id": "50f16d2dc2bb80a421afc1af38fc92e3",
      "distribution": "temurin",
      "java_version": "24.0.2+12",
      "distribution_version": "24.0.2"
    },
    {
      "id": "4b983e5b6800eee4023259bd42e03844",
      "distribution": "temurin",
      "java_version": "24+36",
      "distribution_version": "24"
    }
  ]
}`

func TestGetPackages_QueryParams(t *testing.T) {
	type wantParam struct {
		key, value string
	}
	tests := []struct {
		name   string
		os     string
		arch   string
		dist   string
		ver    string
		expect []wantParam
	}{
		{
			name: "linux amd64 temurin 21",
			os:   "linux",
			arch: "amd64",
			dist: "temurin",
			ver:  "24",
			expect: []wantParam{
				{"operating_system", "linux"},
				{"architecture", "amd64"},
				{"distribution", "temurin"},
				{"version", "24"},
				{"archive_type", "tar.gz"},
				{"lib_c_type", "glibc"},
				{"package_type", "jdk"},
				{"release_status", "ga"},
			},
		},
		{
			name: "windows arm64 zulu 24",
			os:   "windows",
			arch: "arm64",
			dist: "zulu",
			ver:  "24",
			expect: []wantParam{
				{"operating_system", "windows"},
				{"architecture", "arm64"},
				{"distribution", "zulu"},
				{"version", "24"},
				{"archive_type", "zip"},
				{"lib_c_type", "c_std_lib"},
				{"package_type", "jdk"},
				{"release_status", "ga"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotParams url.Values

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotParams = r.URL.Query()
				w.WriteHeader(http.StatusOK)
				io.WriteString(w, mockPackagesResponse)
			}))
			defer server.Close()

			client := &Client{
				BaseURL:    server.URL,
				HTTPClient: server.Client(),
			}

			packages, err := client.GetPackages(tc.os, tc.arch, tc.dist, tc.ver)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Check expected parameters
			for _, want := range tc.expect {
				if got := gotParams.Get(want.key); got != want.value {
					t.Errorf("param %q: got %q, want %q", want.key, got, want.value)
				}
			}

			if len(packages) != 2 {
				t.Fatalf("expected 2 packages, got %d", len(packages))
			}
			if packages[0].Id != "50f16d2dc2bb80a421afc1af38fc92e3" || packages[0].Distribution != "temurin" || packages[0].JavaVersion != "24.0.2+12" || packages[0].DistributionVersion != "24.0.2" {
				t.Errorf("unexpected first package: %+v", packages[0])
			}
			if packages[1].Id != "4b983e5b6800eee4023259bd42e03844" || packages[1].Distribution != "temurin" || packages[1].JavaVersion != "24+36" || packages[1].DistributionVersion != "24" {
				t.Errorf("unexpected second package: %+v", packages[1])
			}
		})
	}
}

const mockPackageInfoResponse = `{
  "result": [
    {
      "filename": "OpenJDK17U-jdk_x64_windows_hotspot_17.0.8_7.zip",
      "direct_download_uri": "https://example.com/download/jdk17.zip",
      "checksum": "abc123def456",
      "checksum_type": "sha256"
    }
  ]
}`

func TestGetPackageInfo(t *testing.T) {
	tests := []struct {
		name         string
		packageID    string
		wantFilename string
		wantError    bool
	}{
		{
			name:         "successful response",
			packageID:    "test-package-id",
			wantFilename: "OpenJDK17U-jdk_x64_windows_hotspot_17.0.8_7.zip",
			wantError:    false,
		},
		{
			name:         "server error",
			packageID:    "error-package-id",
			wantFilename: "",
			wantError:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/ids/error-package-id" {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				w.WriteHeader(http.StatusOK)
				io.WriteString(w, mockPackageInfoResponse)
			}))
			defer server.Close()

			client := &Client{
				BaseURL:    server.URL,
				HTTPClient: server.Client(),
			}

			packageInfo, err := client.GetPackageInfo(tc.packageID)

			if (err != nil) != tc.wantError {
				t.Fatalf("unexpected error condition: got error %v, wantError %v", err, tc.wantError)
			}

			if tc.wantError {
				if packageInfo != nil {
					t.Errorf("expected nil package info when error occurs, got %+v", packageInfo)
				}
				return
			}

			if packageInfo.Filename != tc.wantFilename {
				t.Errorf("unexpected filename: got %q, want %q", packageInfo.Filename, tc.wantFilename)
			}
			if packageInfo.DirectDownloadUri != "https://example.com/download/jdk17.zip" {
				t.Errorf("unexpected download URI: got %q, want %q", packageInfo.DirectDownloadUri, "https://example.com/download/jdk17.zip")
			}
			if packageInfo.Checksum != "abc123def456" {
				t.Errorf("unexpected checksum: got %q, want %q", packageInfo.Checksum, "abc123def456")
			}
			if packageInfo.ChecksumType != "sha256" {
				t.Errorf("unexpected checksum type: got %q, want %q", packageInfo.ChecksumType, "sha256")
			}
		})
	}
}
