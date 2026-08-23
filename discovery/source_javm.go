package discovery

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strings"

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
	jdks, scanErr := s.DiscoverManaged(ctx)
	linked, linkedErr := s.discoverLinkedJDKs(ctx)
	return append(jdks, linked...), errors.Join(scanErr, linkedErr)
}

func (s *JavmSource) DiscoverManaged(ctx context.Context) ([]JDK, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	return ScanLocationsForJDKsContext(ctx, s.root, s.vfs, s.runner, []string{"jdk"}, s.Name())
}

func (s *JavmSource) discoverLinkedJDKs(ctx context.Context) ([]JDK, error) {
	const location = "jdk"

	entries, err := fs.ReadDir(s.vfs, location)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, &DiscoveryWarning{
			Source:   s.Name(),
			Location: location,
			Err:      fmt.Errorf("read directory: %w", err),
		}
	}

	var jdks []JDK
	var warnings []error
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if entry.Type()&fs.ModeSymlink == 0 || !strings.HasPrefix(entry.Name(), "system@") {
			continue
		}

		candidatePath := path.Join(location, entry.Name())
		jdk, ok, err := ValidateJDKContext(ctx, s.vfs, s.runner, s.root, candidatePath, s.Name())
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return nil, ctxErr
			}
			warnings = append(warnings, &DiscoveryWarning{
				Source:   s.Name(),
				Location: location,
				Path:     candidatePath,
				Err:      fmt.Errorf("validate JDK: %w", err),
			})
			continue
		}
		if ok {
			jdks = append(jdks, jdk)
		}
	}

	return jdks, errors.Join(warnings...)
}
