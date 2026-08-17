package discovery

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/felipebz/javm/internal/state"
)

var ErrInvalidCache = errors.New("invalid discovery cache")

type JDK struct {
	Path         string `json:"path"`
	Version      string `json:"version"`
	Vendor       string `json:"vendor"`
	Architecture string `json:"architecture"`
	Source       string `json:"source"`
	Identifier   string `json:"identifier"`
}

// DiscoveryWarning describes a non-fatal failure while discovering JDKs.
// Callers can inspect its fields with errors.As to provide actionable diagnostics.
type DiscoveryWarning struct {
	Source   string
	Location string
	Path     string
	Err      error
}

func (w *DiscoveryWarning) Error() string {
	where := ""
	if w.Path != "" {
		where = fmt.Sprintf(" at path %q", w.Path)
	} else if w.Location != "" {
		where = fmt.Sprintf(" at location %q", w.Location)
	}
	if w.Source != "" {
		return fmt.Sprintf("discover JDKs from source %q%s: %v", w.Source, where, w.Err)
	}
	return fmt.Sprintf("discover JDKs%s: %v", where, w.Err)
}

func (w *DiscoveryWarning) Unwrap() error {
	return w.Err
}

type Cache struct {
	LastUpdated       time.Time `json:"last_updated"`
	ConfigFingerprint string    `json:"config_fingerprint,omitempty"`
	JDKs              []JDK     `json:"jdks"`
}

func (c *Cache) SaveCache(cacheFile string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("encode discovery cache: %w", err)
	}
	data = append(data, '\n')
	return state.WithFileLock(cacheFile, func() error {
		if err := state.AtomicWriteFile(cacheFile, data, 0o600); err != nil {
			return fmt.Errorf("write discovery cache: %w", err)
		}
		return nil
	})
}

func LoadCache(cacheFile string) (*Cache, error) {
	if _, err := os.Stat(cacheFile); os.IsNotExist(err) {
		return &Cache{
			LastUpdated: time.Time{},
			JDKs:        []JDK{},
		}, nil
	}

	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil, err
	}

	var cache Cache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidCache, err)
	}

	return &cache, nil
}

func (c *Cache) IsCacheValid(ttl time.Duration) bool {
	return !c.LastUpdated.IsZero() && time.Since(c.LastUpdated) < ttl
}
