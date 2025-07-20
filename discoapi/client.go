package discoapi

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

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
	fullURL, _ := url.JoinPath(c.BaseURL, endpoint)
	if params != nil && len(params) > 0 {
		fullURL += "?" + params.Encode()
	}

	resp, err := c.HTTPClient.Get(fullURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("GET %s returned %d", fullURL, resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
