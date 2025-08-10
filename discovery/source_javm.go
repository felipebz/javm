package discovery

import (
	"io/fs"
	"os"
	"path"

	"github.com/felipebz/javm/cfg"
)

type JavmSource struct {
	vfs fs.FS
}

func NewJavmSource() *JavmSource {
	return &JavmSource{vfs: os.DirFS("/")}
}

func (s *JavmSource) Name() string {
	return "javm"
}

func (s *JavmSource) Discover() ([]JDK, error) {
	var locations []string

	javmDir := cfg.Dir()
	jdksDir := path.Join(javmDir, "jdk")

	locations = append(locations, jdksDir)

	return ScanLocationsForJDKs(s.vfs, locations, s.Name())
}
