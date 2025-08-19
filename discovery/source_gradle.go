package discovery

import (
	"io/fs"
	"os"
	"path"
)

type GradleSource struct {
	vfs    fs.FS
	runner Runner
}

func NewGradleSource() *GradleSource {
	if gh := os.Getenv("GRADLE_USER_HOME"); gh != "" {
		return &GradleSource{
			vfs:    os.DirFS(gh),
			runner: ExecRunner{},
		}
	}
	return &GradleSource{
		vfs:    os.DirFS(mustHome()),
		runner: ExecRunner{},
	}
}

func (s *GradleSource) Name() string { return "gradle" }

func (s *GradleSource) Discover() ([]JDK, error) {
	locations := []string{
		"jdks",
		path.Join(".gradle", "jdks"),
	}
	return ScanLocationsForJDKs(s.vfs, s.runner, locations, s.Name())
}
