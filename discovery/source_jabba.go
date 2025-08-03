package discovery

import (
	"fmt"
	"os"
	"path/filepath"
)

type JabbaSource struct {
}

func NewJabbaSource() *JabbaSource {
	return &JabbaSource{}
}

func (s *JabbaSource) Name() string {
	return "jabba"
}

func (s *JabbaSource) Enabled(config *Config) bool {
	return config.IsSourceEnabled(s.Name())
}

func (s *JabbaSource) Discover() ([]JDK, error) {
	var locations []string

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	jabbaJdkDir := filepath.Join(homeDir, ".jabba", "jdk")
	if _, err := os.Stat(jabbaJdkDir); err == nil {
		locations = append(locations, jabbaJdkDir)
	}

	return ScanLocationsForJDKs(locations, s.Name())
}
