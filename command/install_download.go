package command

import (
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/schollz/progressbar/v3"
)

const (
	maxArtifactSize = int64(1 << 30) // 1 GiB compressed download
	downloadTimeout = 30 * time.Minute
)

type RedirectTracer struct {
	Transport http.RoundTripper
}

func (r RedirectTracer) RoundTrip(req *http.Request) (*http.Response, error) {
	transport := r.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	resp, err := transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	switch resp.StatusCode {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther, http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
		loggerFromContext(req.Context()).Debug("Following ", resp.StatusCode, " redirect to ", resp.Header.Get("Location"))
	}
	return resp, nil
}

func download(ctx context.Context, rawURL string) (string, error) {
	transport := http.DefaultTransport
	if defaultTransport, ok := http.DefaultTransport.(*http.Transport); ok {
		configured := defaultTransport.Clone()
		configured.ResponseHeaderTimeout = 30 * time.Second
		configured.TLSHandshakeTimeout = 15 * time.Second
		transport = configured
	}
	client := &http.Client{
		Transport:     RedirectTracer{Transport: transport},
		Timeout:       downloadTimeout,
		CheckRedirect: secureRedirect,
	}
	return downloadWithClient(ctx, client, rawURL, maxArtifactSize)
}

func secureRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return fmt.Errorf("download stopped after %d redirects", len(via))
	}
	if req.URL == nil || !strings.EqualFold(req.URL.Scheme, "https") {
		return fmt.Errorf("insecure redirect to non-HTTPS URL: %v", req.URL)
	}
	return nil
}

func downloadWithClient(ctx context.Context, client *http.Client, rawURL string, maxBytes int64) (file string, err error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid download URL: %w", err)
	}
	if !strings.EqualFold(parsedURL.Scheme, "https") {
		return "", fmt.Errorf("insecure download URL: only HTTPS is allowed, got: %s", rawURL)
	}
	if maxBytes <= 0 {
		return "", fmt.Errorf("invalid download size limit: %d", maxBytes)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedURL.String(), nil)
	if err != nil {
		return "", fmt.Errorf("create download request: %w", err)
	}
	res, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download artifact: %w", err)
	}
	defer func() {
		if closeErr := res.Body.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close download response: %w", closeErr))
		}
	}()

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("download artifact returned HTTP %d", res.StatusCode)
	}
	if res.ContentLength > maxBytes {
		return "", fmt.Errorf("download artifact exceeds %d bytes", maxBytes)
	}

	ext := getFileExtension(parsedURL.Path)
	tmp, err := os.CreateTemp("", "javm-download-*"+ext)
	if err != nil {
		return "", fmt.Errorf("create temporary download: %w", err)
	}
	file = tmp.Name()
	tempPath := file
	keep := false
	defer func() {
		closeErr := tmp.Close()
		if err == nil && closeErr != nil {
			err = fmt.Errorf("close temporary download: %w", closeErr)
		}
		if !keep || err != nil {
			if removeErr := os.Remove(tempPath); removeErr != nil && !os.IsNotExist(removeErr) {
				err = errors.Join(err, fmt.Errorf("remove incomplete download: %w", removeErr))
			}
		}
	}()

	runtime := RuntimeFromContext(ctx)
	runtime.Logger.Debug("Saving ", rawURL, " to ", file)
	limited := io.LimitReader(res.Body, maxBytes+1)
	destination := io.Writer(tmp)
	var bar *progressbar.ProgressBar
	if runtime.ShowProgress {
		bar = progressbar.NewOptions64(
			res.ContentLength,
			progressbar.OptionSetWriter(runtime.Err),
			progressbar.OptionSetDescription("downloading"),
			progressbar.OptionShowBytes(true),
			progressbar.OptionSetRenderBlankState(true),
			progressbar.OptionThrottle(0),
		)
		destination = io.MultiWriter(tmp, bar)
	}
	written, err := io.Copy(destination, limited)
	if err != nil {
		return "", fmt.Errorf("save downloaded artifact: %w", err)
	}
	if bar != nil {
		if err := bar.Finish(); err != nil {
			return "", fmt.Errorf("finish download progress: %w", err)
		}
	}
	if written > maxBytes {
		return "", fmt.Errorf("download artifact exceeds %d bytes", maxBytes)
	}
	if err := tmp.Sync(); err != nil {
		return "", fmt.Errorf("sync downloaded artifact: %w", err)
	}
	keep = true
	return file, nil
}

func validateChecksum(path string, expected string, algorithm string) (err error) {
	algorithm = strings.ToLower(strings.TrimSpace(algorithm))
	expected = strings.ToLower(strings.TrimSpace(expected))

	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open artifact for checksum: %w", err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close artifact after checksum: %w", closeErr))
		}
	}()

	var h hash.Hash
	switch algorithm {
	case "sha256":
		h = sha256.New()
	case "sha1":
		h = sha1.New()
	default:
		return fmt.Errorf("unsupported checksum type: %s", algorithm)
	}

	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("calculate %s checksum: %w", algorithm, err)
	}
	actual := fmt.Sprintf("%x", h.Sum(nil))
	if actual != expected {
		return fmt.Errorf("checksum mismatch: expected %s (%s), got %s", expected, algorithm, actual)
	}
	return nil
}

func getFileExtension(file string) string {
	lower := strings.ToLower(file)
	if strings.HasSuffix(lower, ".tar.gz") {
		return ".tar.gz"
	}
	if strings.HasSuffix(lower, ".tar.xz") {
		return ".tar.xz"
	}
	return strings.ToLower(filepath.Ext(file))
}
