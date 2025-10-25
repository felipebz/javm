package discovery

import (
	"io/fs"
	"os"
	"path"
)

type JabbaSource struct {
	root   string
	vfs    fs.FS
	runner Runner
}

func NewJabbaSource() *JabbaSource {
	home := mustHome()
	return &JabbaSource{
		root:   home,
		vfs:    os.DirFS(home),
		runner: ExecRunner{},
	}
}

func (s *JabbaSource) Name() string { return "jabba" }

func (s *JabbaSource) Discover() ([]JDK, error) {
	roots := []string{path.Join(".jabba", "jdk")}
	return ScanLocationsForJDKs(s.root, s.vfs, s.runner, roots, s.Name())
}

func mustHome() string {
	h, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	return h
}
