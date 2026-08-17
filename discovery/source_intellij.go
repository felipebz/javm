package discovery

import (
	"context"
	"io/fs"
	"os"
	"path"
	"runtime"
)

type IntelliJSource struct {
	root   string
	vfs    fs.FS
	runner Runner
}

func NewIntelliJSource() *IntelliJSource {
	home := mustHome()
	return &IntelliJSource{
		root:   home,
		vfs:    os.DirFS(home),
		runner: ExecRunner{},
	}
}

func (s *IntelliJSource) Name() string { return "intellij" }

func (s *IntelliJSource) Discover(ctx context.Context) ([]JDK, error) {
	var locations []string
	if runtime.GOOS == "darwin" {
		locations = append(locations, path.Join("Library", "Java", "JavaVirtualMachines"))
	} else {
		locations = append(locations, ".jdks")
	}
	return ScanLocationsForJDKsContext(ctx, s.root, s.vfs, s.runner, locations, s.Name())
}
