package discoapi

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func fetchStatus(t *testing.T, status int, body string) error {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	client := &Client{BaseURL: server.URL, HTTPClient: server.Client()}
	_, err := client.GetPackages("linux", "amd64", "foo", "")
	if err == nil {
		t.Fatalf("GetPackages() succeeded on HTTP %d", status)
	}
	return err
}

func TestHTTPErrorCarriesStatusAndAPIMessage(t *testing.T) {
	err := fetchStatus(t, http.StatusBadRequest, `{"result":[],"message":"foo is an unsupported distribution"}`)
	httpErr, ok := errors.AsType[*HTTPError](err)
	if !ok || !errors.Is(err, ErrNetwork) {
		t.Fatalf("error = %v, want an *HTTPError matching ErrNetwork", err)
	}
	if httpErr.StatusCode != http.StatusBadRequest || httpErr.Message != "foo is an unsupported distribution" {
		t.Fatalf("HTTPError = %+v, want status 400 and the API message", httpErr)
	}
	if !strings.Contains(err.Error(), "returned 400: foo is an unsupported distribution") {
		t.Fatalf("error text = %q, want the status and the API message", err.Error())
	}
}

func TestHTTPErrorIgnoresUnreadableBodies(t *testing.T) {
	for name, body := range map[string]string{
		"not JSON":       "<html>bad request</html>",
		"message absent": `{"result":[]}`,
		"oversized":      `{"message":"` + strings.Repeat("x", int(maxResponseSize)) + `"}`,
	} {
		t.Run(name, func(t *testing.T) {
			err := fetchStatus(t, http.StatusBadRequest, body)
			httpErr, ok := errors.AsType[*HTTPError](err)
			if !ok || httpErr.StatusCode != http.StatusBadRequest || httpErr.Message != "" {
				t.Fatalf("error = %v, want status 400 without a message", err)
			}
			if !strings.HasSuffix(err.Error(), "returned 400") {
				t.Fatalf("error text = %q, want only the status", err.Error())
			}
		})
	}
}

func TestHTTPErrorOnServerFailure(t *testing.T) {
	err := fetchStatus(t, http.StatusInternalServerError, "")
	httpErr, ok := errors.AsType[*HTTPError](err)
	if !ok || httpErr.StatusCode != http.StatusInternalServerError || !errors.Is(err, ErrNetwork) {
		t.Fatalf("error = %v, want an *HTTPError with status 500 matching ErrNetwork", err)
	}
}

func TestHTTPErrorMessageDropsControlCharacters(t *testing.T) {
	err := fetchStatus(t, http.StatusBadRequest, `{"message":"bad\u001b[2J request\u0007"}`)
	if httpErr, ok := errors.AsType[*HTTPError](err); !ok || httpErr.Message != "bad[2J request" {
		t.Fatalf("error = %v, want the message without control characters", err)
	}
}
