package discovery

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"runtime"
)

type SystemSource struct {
	root      string
	vfs       fs.FS
	runner    Runner
	locations []string
}

type systemRoot struct {
	root     string
	vfs      fs.FS
	location string
}

func NewSystemSource() *SystemSource {
	return &SystemSource{
		root:   "/",
		vfs:    os.DirFS("/"),
		runner: ExecRunner{},
	}
}

func (s *SystemSource) Name() string { return "system" }

func (s *SystemSource) Discover(ctx context.Context) ([]JDK, error) {
	if len(s.locations) > 0 {
		return ScanLocationsForJDKsContext(ctx, s.root, s.vfs, s.runner, s.locations, s.Name())
	}

	return s.discoverRoots(ctx, systemRoots(runtime.GOOS, os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)")))
}

func (s *SystemSource) discoverRoots(ctx context.Context, roots []systemRoot) ([]JDK, error) {
	var all []JDK
	var warnings []error
	for _, root := range roots {
		jdks, err := ScanLocationsForJDKsContext(ctx, root.root, root.vfs, s.runner, []string{root.location}, s.Name())
		all = append(all, jdks...)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return nil, ctxErr
			}
			warnings = append(warnings, err)
		}
	}
	return all, errors.Join(warnings...)
}

func systemRoots(goos, programFiles, programFilesX86 string) []systemRoot {
	switch goos {
	case "linux":
		return []systemRoot{
			{root: "/", vfs: os.DirFS("/"), location: "usr/lib/jvm"},
			{root: "/", vfs: os.DirFS("/"), location: "opt/java"},
		}
	case "darwin":
		return []systemRoot{
			{root: "/", vfs: os.DirFS("/"), location: "Library/Java/JavaVirtualMachines"},
		}
	case "windows":
		var roots []systemRoot
		if programFiles != "" {
			roots = append(roots, systemRoot{root: programFiles, vfs: os.DirFS(programFiles), location: "Java"})
		}
		if programFilesX86 != "" {
			roots = append(roots, systemRoot{root: programFilesX86, vfs: os.DirFS(programFilesX86), location: "Java"})
		}
		return roots
	default:
		return nil
	}
}
