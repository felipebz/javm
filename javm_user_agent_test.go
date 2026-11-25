package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"runtime/debug"
	"testing"

	"github.com/felipebz/javm/internal/useragent"
)

func TestApplicationSharesOneUserAgent(t *testing.T) {
	info := &debug.BuildInfo{Main: debug.Module{Version: "v1.2.3"}}
	app := newApplication(&bytes.Buffer{}, &bytes.Buffer{}, info)
	want := useragent.Format(releaseVersion(version, info))
	if app.userAgent != want || app.client.UserAgent != want {
		t.Fatalf("runtime User-Agent %q, Disco API User-Agent %q, want both %q", app.userAgent, app.client.UserAgent, want)
	}
}

func TestRootCommandSendsUserAgentToDiscoAPI(t *testing.T) {
	isolatedWorkspace(t)
	var got string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("User-Agent")
		_, _ = w.Write([]byte(`{"result":[{"name":"Temurin","api_parameter":"temurin"}]}`))
	}))
	t.Cleanup(server.Close)
	t.Setenv("JAVM_DISCO_API", server.URL)

	var stdout, stderr bytes.Buffer
	info := &debug.BuildInfo{Main: debug.Module{Version: "v1.2.3"}}
	root := newRootCommand(newApplication(&stdout, &stderr, info))
	root.SetArgs([]string{"ls-distributions"})
	if err := root.Execute(); err != nil {
		t.Fatalf("ls-distributions: %v (%s)", err, stderr.String())
	}
	if want := useragent.Format(releaseVersion(version, info)); got != want {
		t.Fatalf("Disco API received User-Agent %q, want %q", got, want)
	}
}
