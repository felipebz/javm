package discoapi

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Sample JSON response similar to what DiscoAPI would return
const mockDistributionsResponse = `{
  "result": [
    {
      "name": "Zulu",
      "api_parameter": "zulu"
    },
    {
      "name": "Temurin",
      "api_parameter": "temurin"
    }
  ]
}`

func TestGetDistributions(t *testing.T) {
	// Set up a fake HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Test that the endpoint and query param are correct
		if r.URL.Path == "/distributions" {
			if r.URL.Query().Get("include_versions") == "false" {
				w.WriteHeader(http.StatusOK)
				io.WriteString(w, mockDistributionsResponse)
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := &Client{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	}

	dists, err := client.GetDistributions()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(dists) != 2 {
		t.Fatalf("expected 2 distributions, got %d", len(dists))
	}
	if dists[0].Name != "Temurin" || dists[0].APIParameter != "temurin" {
		t.Errorf("unexpected first distribution: %+v", dists[0])
	}
	if dists[1].Name != "Zulu" || dists[1].APIParameter != "zulu" {
		t.Errorf("unexpected second distribution: %+v", dists[1])
	}
}
