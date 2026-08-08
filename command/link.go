package command

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/felipebz/javm/cfg"
	"github.com/felipebz/javm/discovery"
	"github.com/felipebz/javm/semver"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func NewLinkCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "link [name] [path]",
		Short: "Resolve or update a link",
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
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return pflag.ErrHelp
			}
			if err := link(cmd.Context(), args[0], ""); err != nil {
				return err
			}
			return nil
		},
		Example: "  javm unlink system@1.8.20",
	}
}

func link(ctx context.Context, selector string, dir string) error {
	if !strings.HasPrefix(selector, "system@") {
		return errors.New("Name must begin with 'system@' (e.g. 'system@1.8.73')")
	}
	// <version> has to be valid per semver
	if _, err := semver.ParseVersion(selector); err != nil {
		return err
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
	files, _ := readDir(filepath.Join(cfg.Dir(), "jdk"))
	var jdks, err = Ls(true)
	if err != nil {
		return err
	}
	cache := make(map[string]string)
	for _, f := range files {
		info, _ := f.Info()
		if f.IsDir() || info.Mode()&os.ModeSymlink == os.ModeSymlink {
			sourceVersion := f.Name()
			if strings.Count(sourceVersion, ".") == 1 && !strings.HasPrefix(sourceVersion, "system@") {
				target := getLink(sourceVersion)
				_, err := FindBestMatchJDK(jdks, sourceVersion)
				if err != nil {
					err := os.Remove(filepath.Join(cfg.Dir(), "jdk", sourceVersion))
					if err == nil {
						loggerFromContext(ctx).Info(sourceVersion + " -/> " + target)
					}
					if !os.IsNotExist(err) {
						return err
					}
				} else {
					cache[sourceVersion] = target
				}
			}
		}
	}

	// Convert discovery.JDK to semver.Version for sorting/trimming
	var versions []*semver.Version
	for _, jdk := range jdks {
		if v, err := semver.ParseVersion(jdk.Identifier); err == nil {
			versions = append(versions, v)
		} else if v, err := semver.ParseVersion(jdk.Version); err == nil {
			// fallback check
			versions = append(versions, v)
		}
	}

	for _, v := range semver.VersionSlice(versions).TrimTo(semver.VPMinor) {
		sourceVersion := v.TrimTo(semver.VPMinor)
		target := filepath.Join(cfg.Dir(), "jdk", v.String())
		if v.Prerelease() == "" && cache[sourceVersion] != target && !strings.HasPrefix(sourceVersion, "system@") {
			source := filepath.Join(cfg.Dir(), "jdk", sourceVersion)
			loggerFromContext(ctx).Info(v.String() + " -> " + target)
			os.Remove(source)
			if err := os.Symlink(target, source); err != nil {
				return err
			}
		}
	}
	return linkAlias(ctx, "default", jdks)
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
			os.Remove(source)
			if err := os.Symlink(target, source); err != nil {
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
