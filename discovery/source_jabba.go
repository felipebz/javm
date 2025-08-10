package discovery

import (
	"io/fs"
	"os"
	"path"
)

type JabbaSource struct {
	vfs    fs.FS
	runner Runner
}

func NewJabbaSource() *JabbaSource {
	return &JabbaSource{
		vfs:    os.DirFS(mustHome()),
		runner: ExecRunner{},
	}
}

func (s *JabbaSource) Name() string { return "jabba" }

func (s *JabbaSource) Discover() ([]JDK, error) {
	roots := []string{path.Join(".jabba", "jdk")}
	return ScanLocationsForJDKs(s.vfs, s.runner, roots, s.Name())
}

func mustHome() string {
	h, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	return h
}
