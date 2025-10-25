package discovery

import (
	"io/fs"
	"os"

	"github.com/felipebz/javm/cfg"
)

type JavmSource struct {
	root   string
	vfs    fs.FS
	runner Runner
}

func NewJavmSource() *JavmSource {
	dir := cfg.Dir()
	return &JavmSource{
		root:   dir,
		vfs:    os.DirFS(dir),
		runner: ExecRunner{},
	}
}

func (s *JavmSource) Name() string {
	return "javm"
}

func (s *JavmSource) Discover() ([]JDK, error) {
	return ScanLocationsForJDKs(s.root, s.vfs, s.runner, []string{"jdk"}, s.Name())
}
