package discoapi

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestClientFetch(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/some-endpoint":
			if r.URL.RawQuery == "foo=bar" {
				w.WriteHeader(http.StatusOK)
				io.WriteString(w, `{"ok":"with-query"}`)
			} else if r.URL.RawQuery == "" {
				w.WriteHeader(http.StatusOK)
				io.WriteString(w, `{"ok":"no-query"}`)
			} else {
				w.WriteHeader(http.StatusBadRequest)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}

	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	client := &Client{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	}

	tests := []struct {
		name     string
		endpoint string
		params   url.Values
		wantBody string
		wantErr  bool
	}{
		{
			name:     "fetch with no query",
			endpoint: "some-endpoint",
			params:   nil,
			wantBody: `{"ok":"no-query"}`,
			wantErr:  false,
		},
		{
			name:     "fetch with query",
			endpoint: "some-endpoint",
			params:   url.Values{"foo": []string{"bar"}},
			wantBody: `{"ok":"with-query"}`,
			wantErr:  false,
		},
		{
			name:     "not found",
			endpoint: "other-endpoint",
			params:   nil,
			wantBody: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := client.fetch(tt.endpoint, tt.params)
			if (err != nil) != tt.wantErr {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tt.wantErr && string(body) != tt.wantBody {
				t.Errorf("expected body %q, got %q", tt.wantBody, string(body))
			}
		})
	}
}
