package command

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/felipebz/javm/cfg"
	"github.com/felipebz/javm/discovery"
	"github.com/felipebz/javm/javaversion"
	"github.com/spf13/cobra"
)

var removePath = os.Remove

func replaceSymlink(target, source string) error {
	err := removePath(source)
	switch {
	case err == nil:
	case errors.Is(err, fs.ErrNotExist):
	default:
		return fmt.Errorf("remove existing symlink %q: %w", source, err)
	}
	if err := os.Symlink(target, source); err != nil {
		return fmt.Errorf("create symlink %q -> %q: %w", source, target, err)
	}
	return nil
}

func NewLinkCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "link [name] [path]",
		Short: "Resolve or update a link",
		Args:  UsageArgs(cobra.RangeArgs(0, 2)),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				if err := linkLatest(cmd.Context()); err != nil {
					return err
				}
				return nil
			}
			if len(args) == 1 {
				if value := getLink(args[0]); value != "" {
					if _, err := fmt.Fprintln(cmd.OutOrStdout(), value); err != nil {
						return fmt.Errorf("write link: %w", err)
					}
				}
			} else if err := link(cmd.Context(), args[0], args[1]); err != nil {
				return err
			}
			return nil
		},
		Example: "  javm link system@1.8.20 /Library/Java/JavaVirtualMachines/jdk1.8.0_20.jdk\n" +
			"  javm link system@1.8.20 # show link target",
	}
}

func NewUnlinkCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "unlink [name]",
		Short: "Delete a link",
		Args:  UsageArgs(cobra.ExactArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := link(cmd.Context(), args[0], ""); err != nil {
				return err
			}
			return nil
		},
		Example: "  javm unlink system@1.8.20",
	}
}

func link(ctx context.Context, selector string, dir string) error {
	if !isSystemSelector(selector) {
		return UsageError(errors.New("Name must begin with 'system@' (e.g. 'system@1.8.73')"))
	}
	if _, err := javaversion.ParseVersion(selector); err != nil {
		return UsageError(err)
	}
	if dir == "" {
		ver, err := LsBestMatchContext(ctx, selector, false)
		if err != nil {
			return err
		}
		if err := os.Remove(filepath.Join(cfg.Dir(), "jdk", ver)); err != nil {
			return err
		}
	} else {
		if err := assertJavaDistribution(dir, runtime.GOOS); err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Join(cfg.Dir(), "jdk"), 0755); err != nil {
			return err
		}
		if err := os.Symlink(dir, filepath.Join(cfg.Dir(), "jdk", selector)); err != nil {
			return err
		}
	}
	return invalidateDiscoveryCache()
}

func linkLatest(ctx context.Context) (resultErr error) {
	defer func() {
		if err := invalidateDiscoveryCache(); err != nil {
			if resultErr == nil {
				resultErr = err
			} else {
				loggerFromContext(ctx).Warn(err)
			}
		}
	}()
	jdks, err := LsContext(ctx, true)
	if err != nil {
		return err
	}
	links, err := pruneStaleInterimLinks(ctx, jdks)
	if err != nil {
		return err
	}
	for _, v := range javaversion.VersionSlice(managedVersions(jdks)).TrimTo(javaversion.VPInterim) {
		name := v.TrimTo(javaversion.VPInterim)
		target := filepath.Join(cfg.Dir(), "jdk", v.String())
		if v.Prerelease() != "" || links[name] == target || isSystemSelector(name) {
			continue
		}
		loggerFromContext(ctx).Info(v.String() + " -> " + target)
		if err := replaceSymlink(target, filepath.Join(cfg.Dir(), "jdk", name)); err != nil {
			return err
		}
	}
	return linkAlias(ctx, "default", jdks)
}

// pruneStaleInterimLinks removes jdk/<dist@feature.interim> entries that no
// installed JDK matches anymore and returns the current target of the others.
func pruneStaleInterimLinks(ctx context.Context, jdks []discovery.JDK) (map[string]string, error) {
	files, err := readDir(filepath.Join(cfg.Dir(), "jdk"))
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return map[string]string{}, nil
	case err != nil:
		return nil, fmt.Errorf("read managed JDK directory: %w", err)
	}
	links := make(map[string]string)
	for _, f := range files {
		name := f.Name()
		if strings.Count(name, ".") != 1 || isSystemSelector(name) || (!f.IsDir() && f.Type()&os.ModeSymlink == 0) {
			continue
		}
		target := getLink(name)
		if _, err := FindBestMatchJDK(jdks, name); err == nil {
			links[name] = target
			continue
		}
		err := os.Remove(filepath.Join(cfg.Dir(), "jdk", name))
		if err == nil {
			loggerFromContext(ctx).Info(name + " -/> " + target)
		} else if !os.IsNotExist(err) {
			return nil, err
		}
	}
	return links, nil
}

// managedVersions parses each JDK identifier, falling back to its version.
func managedVersions(jdks []discovery.JDK) []*javaversion.Version {
	var versions []*javaversion.Version
	for _, jdk := range jdks {
		if v, err := javaversion.ParseVersion(jdk.Identifier); err == nil {
			versions = append(versions, v)
		} else if v, err := javaversion.ParseVersion(jdk.Version); err == nil {
			versions = append(versions, v)
		}
	}
	return versions
}

func isSystemSelector(name string) bool {
	return strings.HasPrefix(name, "system@")
}

func linkAliasName(ctx context.Context, name string) error {
	if err := validateAliasName(name); err != nil {
		return err
	}
	var jdks, err = LsContext(ctx, false)
	if err != nil {
		return err
	}
	if err := linkAlias(ctx, name, jdks); err != nil {
		return err
	}
	return invalidateDiscoveryCache()
}

func invalidateDiscoveryCache() error {
	if err := discovery.DeleteCacheFile(cfg.Dir()); err != nil {
		return fmt.Errorf("invalidate discovery cache: %w", err)
	}
	return nil
}

func linkAlias(ctx context.Context, name string, jdks []discovery.JDK) error {
	if err := validateAliasName(name); err != nil {
		return err
	}
	defaultAlias := getAlias(name)
	if defaultAlias != "" {
		if jdk, err := FindBestMatchJDK(jdks, defaultAlias); err == nil {
			defaultAlias = jdk.Identifier
		}
	}
	sourceRef := /*"alias@" + */ name
	source := filepath.Join(cfg.Dir(), "jdk", sourceRef)
	sourceTarget := getLink(sourceRef)
	if defaultAlias != "" {
		target := filepath.Join(cfg.Dir(), "jdk", defaultAlias)
		if sourceTarget != target {
			loggerFromContext(ctx).Info(sourceRef + " -> " + target)
			if err := replaceSymlink(target, source); err != nil {
				return err
			}
		}
	} else {
		err := os.Remove(source)
		if err == nil {
			loggerFromContext(ctx).Info(sourceRef + " -/> " + sourceTarget)
		}
		if !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func getLink(name string) string {
	res, err := filepath.EvalSymlinks(filepath.Join(cfg.Dir(), "jdk", name))
	if err != nil {
		return ""
	}
	return res
}
