package discovery

import (
	"context"
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

func (s *JavmSource) Discover(ctx context.Context) ([]JDK, error) {
	return ScanLocationsForJDKsContext(ctx, s.root, s.vfs, s.runner, []string{"jdk"}, s.Name())
}
