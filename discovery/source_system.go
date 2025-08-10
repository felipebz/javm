package discovery

import (
	"io/fs"
	"os"
	"runtime"
)

type SystemSource struct {
	vfs       fs.FS
	runner    Runner
	locations []string
}

func NewSystemSource() *SystemSource {
	return &SystemSource{
		vfs:    os.DirFS("/"),
		runner: ExecRunner{},
	}
}

func (s *SystemSource) Name() string { return "system" }

func (s *SystemSource) Discover() ([]JDK, error) {
	if len(s.locations) > 0 {
		return ScanLocationsForJDKs(s.vfs, s.runner, s.locations, s.Name())
	}

	type root struct {
		vfs  fs.FS
		path string
	}
	var roots []root

	switch runtime.GOOS {
	case "linux":
		roots = append(roots,
			root{vfs: os.DirFS("/"), path: "usr/lib/jvm"},
			root{vfs: os.DirFS("/"), path: "opt/java"},
		)
	case "darwin":
		roots = append(roots,
			root{vfs: os.DirFS("/"), path: "Library/Java/JavaVirtualMachines"},
		)
	case "windows":
		if pf := os.Getenv("ProgramFiles"); pf != "" {
			roots = append(roots, root{vfs: os.DirFS(pf), path: "Java"})
		}
		if pf86 := os.Getenv("ProgramFiles(x86)"); pf86 != "" {
			roots = append(roots, root{vfs: os.DirFS(pf86), path: "Java"})
		}
	}

	var all []JDK
	for _, r := range roots {
		jdks, err := ScanLocationsForJDKs(r.vfs, s.runner, []string{r.path}, s.Name())
		if err != nil {
			return nil, err
		}
		all = append(all, jdks...)
	}
	return all, nil
}
