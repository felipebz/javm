package discovery

import (
	"path/filepath"

	"github.com/felipebz/javm/cfg"
)

type JavmSource struct {
}

func NewJavmSource() *JavmSource {
	return &JavmSource{}
}

func (s *JavmSource) Name() string {
	return "javm"
}

func (s *JavmSource) Discover() ([]JDK, error) {
	var locations []string

	javmDir := cfg.Dir()
	jdksDir := filepath.Join(javmDir, "jdk")

	locations = append(locations, jdksDir)

	return ScanLocationsForJDKs(locations, s.Name())
}
