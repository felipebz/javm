package discovery

import (
	"io/fs"
	"os"
	"path"
	"runtime"
)

type IntelliJSource struct {
	vfs    fs.FS
	runner Runner
}

func NewIntelliJSource() *IntelliJSource {
	return &IntelliJSource{
		vfs:    os.DirFS(mustHome()),
		runner: ExecRunner{},
	}
}

func (s *IntelliJSource) Name() string { return "intellij" }

func (s *IntelliJSource) Discover() ([]JDK, error) {
	var locations []string
	if runtime.GOOS == "darwin" {
		locations = append(locations, path.Join("Library", "Java", "JavaVirtualMachines"))
	} else {
		locations = append(locations, ".jdks")
	}
	return ScanLocationsForJDKs(s.vfs, s.runner, locations, s.Name())
}
