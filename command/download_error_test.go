package command

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDownloadErrorsNameThePrefixOnce(t *testing.T) {
	unavailable := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(unavailable.Close)
	closed := httptest.NewTLSServer(http.NotFoundHandler())
	closedClient, closedURL := closed.Client(), closed.URL
	closed.Close()

	for name, tt := range map[string]struct {
		client *http.Client
		url    string
		want   string
	}{
		"unreachable": {client: closedClient, url: closedURL + "/jdk.tar.gz", want: closedURL},
		"HTTP 503":    {client: unavailable.Client(), url: unavailable.URL + "/jdk.tar.gz", want: "HTTP 503"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := downloadWithClient(context.Background(), tt.client, tt.url, "", 1<<20)
			if !errors.Is(err, ErrNetwork) {
				t.Fatalf("error = %v, want a network error", err)
			}
			if message := err.Error(); strings.Count(message, "network error") != 1 || !strings.Contains(message, tt.want) {
				t.Fatalf("error = %q, want one network error prefix and %q", message, tt.want)
			}
		})
	}
}
