package main

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode"

	"github.com/felipebz/javm/command"
	"github.com/felipebz/javm/discoapi"
	log "github.com/sirupsen/logrus"
)

func executeRootError(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	logger := log.New()
	logger.SetFormatter(&simpleFormatter{})
	root := newRootCommand(application{out: &stdout, err: &stderr, logger: logger, client: discoapi.NewClient()})
	root.SetArgs(args)
	err := root.Execute()
	return stderr.String(), err
}

func hasTerminalControl(s string) bool {
	return strings.ContainsFunc(s, func(r rune) bool {
		return r != '\n' && (r < 0x20 || r == 0x7f || r >= 0x80 && r <= 0x9f)
	})
}

func TestNetworkErrorsNameThePrefixAndURLOnce(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	url := server.URL
	server.Close()

	for _, args := range [][]string{{"ls-remote"}, {"install", "21"}, {"ls-distributions"}} {
		t.Run(args[0], func(t *testing.T) {
			isolatedWorkspace(t)
			t.Setenv(discoapi.EnvVar, url)
			stderr, err := executeRootError(t, args...)
			if exitCode(err) != exitNetwork || !errors.Is(err, command.ErrNetwork) || !errors.Is(err, discoapi.ErrNetwork) {
				t.Fatalf("error = %v, exit %d, want a network error from both layers", err, exitCode(err))
			}
			if strings.Count(stderr, "network error") != 1 || strings.Count(stderr, url) != 1 {
				t.Fatalf("stderr = %q, want %q and %q once each", stderr, "network error", url)
			}
		})
	}
}

func TestNetworkErrorKeepsHTTPStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)
	isolatedWorkspace(t)
	t.Setenv(discoapi.EnvVar, server.URL)

	stderr, err := executeRootError(t, "ls-remote")
	if exitCode(err) != exitNetwork || !errors.Is(err, discoapi.ErrNetwork) {
		t.Fatalf("error = %v, exit %d, want a network error", err, exitCode(err))
	}
	if !strings.Contains(stderr, "returned 503") || strings.Count(stderr, "network error") != 1 {
		t.Fatalf("stderr = %q, want the 503 status and one network error prefix", stderr)
	}
}

func packagesServer(t *testing.T, body string) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	t.Setenv(discoapi.EnvVar, server.URL)
}

func TestErrorMessagesStartLowercase(t *testing.T) {
	tests := [][]string{
		{"install", "99"},
		{"link", "foo", "/nonexistent"},
		{"uninstall", "system@21"},
	}
	for _, args := range tests {
		t.Run(args[0], func(t *testing.T) {
			isolatedWorkspace(t)
			packagesServer(t, `{"result":[{"id":"a","distribution":"temurin","java_version":"21.0.1","distribution_version":"21.0.1"}]}`)
			stderr, err := executeRootError(t, args...)
			if err == nil {
				t.Fatal("expected an error")
			}
			message := strings.TrimPrefix(stderr, "Error: ")
			for _, prefix := range []string{"usage error: ", "not found: "} {
				message = strings.TrimPrefix(message, prefix)
			}
			if first := []rune(message); len(first) == 0 || !unicode.IsLower(first[0]) {
				t.Fatalf("stderr = %q, want the message to start lowercase", stderr)
			}
		})
	}
}

func TestErrorMessagesStripControlCharacters(t *testing.T) {
	tests := [][]string{
		{"install", "99\v"},
		{"install", "21\x1b[31m"},
		{"uninstall", "system@x\x1b[31m"},
	}
	for _, args := range tests {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			isolatedWorkspace(t)
			packagesServer(t, `{"result":[`+
				`{"id":"a","distribution":"temurin","java_version":"21.0.1","distribution_version":"21.0.1"},`+
				`{"id":"b","distribution":"temurin\u001b[31m","java_version":"21.0.2\u001b[31m","distribution_version":"21.0.2"}]}`)
			stderr, err := executeRootError(t, args...)
			if err == nil || stderr == "" {
				t.Fatalf("error = %v, stderr = %q, want a printed error", err, stderr)
			}
			if hasTerminalControl(stderr) {
				t.Fatalf("stderr = %q, want no control characters", stderr)
			}
		})
	}
}
