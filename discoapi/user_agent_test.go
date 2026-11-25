package discoapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchSendsUserAgent(t *testing.T) {
	var got []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = append(got, r.Header.Get("User-Agent"))
		if r.URL.Query().Get("distribution") == "foo" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_, _ = w.Write([]byte(`{"result":[]}`))
	}))
	t.Cleanup(server.Close)

	for _, tc := range []struct {
		name, userAgent, want string
	}{
		{"configured", "javm/9.9.9 (+https://github.com/felipebz/javm)", "javm/9.9.9 (+https://github.com/felipebz/javm)"},
		{"client literal", "", "javm/dev (+https://github.com/felipebz/javm)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got = nil
			client := &Client{BaseURL: server.URL, HTTPClient: server.Client(), UserAgent: tc.userAgent}
			if _, err := client.GetPackages("linux", "amd64", "temurin", ""); err != nil {
				t.Fatal(err)
			}
			if _, err := client.GetPackages("linux", "amd64", "foo", ""); err == nil {
				t.Fatal("GetPackages succeeded on HTTP 400")
			}
			if len(got) != 2 || got[0] != tc.want || got[1] != tc.want {
				t.Fatalf("User-Agent headers = %q, want %q on the success and the error request", got, tc.want)
			}
		})
	}
}
