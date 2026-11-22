package discoapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
	"unicode"

	log "github.com/sirupsen/logrus"
)

const maxResponseSize int64 = 10 << 20

var ErrNetwork = errors.New("network error")

// networkError matches ErrNetwork without adding its text: the command layer
// prints the "network error" prefix once.
type networkError struct{ error }

func (e networkError) Is(target error) bool { return target == ErrNetwork }
func (e networkError) Unwrap() error        { return e.error }

// HTTPError is a non-2xx answer from the Disco API. It matches ErrNetwork, so
// callers and exit codes treat it as before; StatusCode and Message let a
// caller look closer at a rejected request.
type HTTPError struct {
	URL        string
	StatusCode int
	// Message is the API's "message" field, without control characters, or
	// "" when the body is not such a JSON object.
	Message string
}

func (e *HTTPError) Error() string {
	text := fmt.Sprintf("GET %s returned %d", e.URL, e.StatusCode)
	if e.Message != "" {
		text += ": " + e.Message
	}
	return text
}

func (e *HTTPError) Is(target error) bool { return target == ErrNetwork }

// errorMessage reads the "message" of an error body within maxResponseSize.
func errorMessage(body io.Reader) string {
	data, err := io.ReadAll(io.LimitReader(body, maxResponseSize+1))
	if err != nil || int64(len(data)) > maxResponseSize {
		return ""
	}
	var response struct {
		Message string `json:"message"`
	}
	if json.Unmarshal(data, &response) != nil {
		return ""
	}
	return strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, response.Message))
}

const (
	DefaultDiscoAPIURL = "https://api.foojay.io/disco/v3.0"
	EnvVar             = "JAVM_DISCO_API"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	Logger     *log.Logger
}

func NewClient() *Client {
	apiURL := os.Getenv(EnvVar)
	if apiURL == "" {
		apiURL = DefaultDiscoAPIURL
	}

	return &Client{
		BaseURL: apiURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		Logger: log.New(),
	}
}

func (c *Client) logger() *log.Logger {
	if c.Logger != nil {
		return c.Logger
	}
	logger := log.New()
	logger.SetOutput(io.Discard)
	return logger
}

func (c *Client) fetch(endpoint string, params url.Values) ([]byte, error) {
	return c.fetchContext(context.Background(), endpoint, params)
}

func (c *Client) fetchContext(ctx context.Context, endpoint string, params url.Values) (data []byte, err error) {
	fullURL, err := url.JoinPath(c.BaseURL, endpoint)
	if err != nil {
		return nil, networkError{fmt.Errorf("build DiscoAPI URL: %w", err)}
	}
	if params != nil && len(params) > 0 {
		fullURL += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, networkError{fmt.Errorf("build GET %s: %w", fullURL, err)}
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		// *url.Error already names the method and URL.
		return nil, networkError{err}
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			err = errors.Join(err, networkError{fmt.Errorf("close GET %s response: %w", fullURL, closeErr)})
		}
	}()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, &HTTPError{URL: fullURL, StatusCode: resp.StatusCode, Message: errorMessage(resp.Body)}
	}

	if resp.ContentLength > maxResponseSize {
		return nil, networkError{fmt.Errorf("GET %s response exceeds %d bytes", fullURL, maxResponseSize)}
	}
	data, err = io.ReadAll(io.LimitReader(resp.Body, maxResponseSize+1))
	if err != nil {
		return nil, networkError{fmt.Errorf("read GET %s response: %w", fullURL, err)}
	}
	if int64(len(data)) > maxResponseSize {
		return nil, networkError{fmt.Errorf("GET %s response exceeds %d bytes", fullURL, maxResponseSize)}
	}
	return data, nil
}
