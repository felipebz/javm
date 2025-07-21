package discoapi

import (
	"io"
	"net/http"
	"net/http/httptest"
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

func TestGetPackages(t *testing.T) {
	// Set up a fake HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Test that the endpoint and query param are correct
		if r.URL.Path == "/packages" {
			//query := r.URL.Query()
			//if query.Get("include_versions") == "false" {
			w.WriteHeader(http.StatusOK)
			io.WriteString(w, mockPackagesResponse)
			return
			//}
			//w.WriteHeader(http.StatusBadRequest)
			//return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := &Client{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	}

	packages, err := client.GetPackages("linux", "x64", "temurin", "24")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
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
}
