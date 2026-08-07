package discoapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

const maxResponseSize int64 = 10 << 20

const (
	DefaultDiscoAPIURL = "https://api.foojay.io/disco/v3.0"
	EnvVar             = "JAVM_DISCO_API"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
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
	}
}

func (c *Client) fetch(endpoint string, params url.Values) ([]byte, error) {
	return c.fetchContext(context.Background(), endpoint, params)
}

func (c *Client) fetchContext(ctx context.Context, endpoint string, params url.Values) (data []byte, err error) {
	fullURL, err := url.JoinPath(c.BaseURL, endpoint)
	if err != nil {
		return nil, fmt.Errorf("build DiscoAPI URL: %w", err)
	}
	if params != nil && len(params) > 0 {
		fullURL += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build GET %s: %w", fullURL, err)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", fullURL, err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close GET %s response: %w", fullURL, closeErr))
		}
	}()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("GET %s returned %d", fullURL, resp.StatusCode)
	}

	if resp.ContentLength > maxResponseSize {
		return nil, fmt.Errorf("GET %s response exceeds %d bytes", fullURL, maxResponseSize)
	}
	data, err = io.ReadAll(io.LimitReader(resp.Body, maxResponseSize+1))
	if err != nil {
		return nil, fmt.Errorf("read GET %s response: %w", fullURL, err)
	}
	if int64(len(data)) > maxResponseSize {
		return nil, fmt.Errorf("GET %s response exceeds %d bytes", fullURL, maxResponseSize)
	}
	return data, nil
}
