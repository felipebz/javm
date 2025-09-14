package discovery

import (
	"io/fs"
	"os"

	"github.com/felipebz/javm/cfg"
)

type JavmSource struct {
	vfs    fs.FS
	runner Runner
}

func NewJavmSource() *JavmSource {
	return &JavmSource{
		vfs:    os.DirFS(cfg.Dir()),
		runner: ExecRunner{},
	}
}

func (s *JavmSource) Name() string {
	return "javm"
}

func (s *JavmSource) Discover() ([]JDK, error) {
	return ScanLocationsForJDKs(s.vfs, s.runner, []string{"jdk"}, s.Name())
}
