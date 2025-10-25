package discovery

import (
	"io/fs"
	"os"
	"path"
)

type GradleSource struct {
	root   string
	vfs    fs.FS
	runner Runner
}

func NewGradleSource() *GradleSource {
	if gh := os.Getenv("GRADLE_USER_HOME"); gh != "" {
		return &GradleSource{
			root:   gh,
			vfs:    os.DirFS(gh),
			runner: ExecRunner{},
		}
	}
	home := mustHome()
	return &GradleSource{
		root:   home,
		vfs:    os.DirFS(home),
		runner: ExecRunner{},
	}
}

func (s *GradleSource) Name() string { return "gradle" }

func (s *GradleSource) Discover() ([]JDK, error) {
	locations := []string{
		"jdks",
		path.Join(".gradle", "jdks"),
	}
	return ScanLocationsForJDKs(s.root, s.vfs, s.runner, locations, s.Name())
}
