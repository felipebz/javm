package command

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
)

type agentRecorder struct {
	mu     sync.Mutex
	agents map[string]string
}

func (r *agentRecorder) record(path, agent string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.agents[path] = agent
}

func TestDownloadSendsUserAgentOnEveryHop(t *testing.T) {
	recorder := &agentRecorder{}
	cdn := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder.record("cdn"+r.URL.Path, r.Header.Get("User-Agent"))
		_, _ = io.WriteString(w, "archive")
	}))
	t.Cleanup(cdn.Close)
	origin := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder.record(r.URL.Path, r.Header.Get("User-Agent"))
		switch r.URL.Path {
		case "/a":
			http.Redirect(w, r, "/b", http.StatusFound)
		case "/b":
			_, _ = io.WriteString(w, "archive")
		case "/other-host":
			http.Redirect(w, r, cdn.URL+"/jdk.zip", http.StatusFound)
		}
	}))
	t.Cleanup(origin.Close)

	client := origin.Client()
	transport := client.Transport.(*http.Transport).Clone()
	transport.TLSClientConfig.RootCAs.AddCert(cdn.Certificate())
	client = &http.Client{Transport: transport, CheckRedirect: secureRedirect}

	for _, tc := range []struct {
		name string
		ctx  context.Context
		want string
	}{
		{"runtime", WithRuntime(context.Background(), Runtime{UserAgent: "javm/9.9.9 (+https://github.com/felipebz/javm)"}), "javm/9.9.9 (+https://github.com/felipebz/javm)"},
		{"no runtime", context.Background(), "javm/dev (+https://github.com/felipebz/javm)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder.agents = map[string]string{}
			for _, path := range []string{"/a", "/other-host"} {
				file, err := downloadWithClient(tc.ctx, client, origin.URL+path, "jdk.zip", 32)
				if err != nil {
					t.Fatalf("download %s: %v", path, err)
				}
				_ = os.Remove(file)
			}
			for _, hop := range []string{"/a", "/b", "/other-host", "cdn/jdk.zip"} {
				if got := recorder.agents[hop]; got != tc.want {
					t.Errorf("hop %s got User-Agent %q, want %q", hop, got, tc.want)
				}
			}
		})
	}
}
