package command

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/felipebz/javm/cfg"
	"github.com/felipebz/javm/discovery"
	"github.com/felipebz/javm/semver"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func NewLinkCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "link [name] [path]",
		Short: "Resolve or update a link",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				if err := linkLatest(); err != nil {
					return err
				}
				return nil
			}
			if len(args) == 1 {
				if value := getLink(args[0]); value != "" {
					fmt.Println(value)
				}
			} else if err := link(args[0], args[1]); err != nil {
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
			if err := link(args[0], ""); err != nil {
				return err
			}
			return nil
		},
		Example: "  javm unlink system@1.8.20",
	}
}

func link(selector string, dir string) error {
	if !strings.HasPrefix(selector, "system@") {
		return errors.New("Name must begin with 'system@' (e.g. 'system@1.8.73')")
	}
	// <version> has to be valid per semver
	if _, err := semver.ParseVersion(selector); err != nil {
		return err
	}
	if dir == "" {
		ver, err := LsBestMatch(selector, false)
		if err != nil {
			return err
		}
		return os.Remove(filepath.Join(cfg.Dir(), "jdk", ver))
	} else {
		if err := assertJavaDistribution(dir, runtime.GOOS); err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Join(cfg.Dir(), "jdk"), 0755); err != nil {
			return err
		}
		return os.Symlink(dir, filepath.Join(cfg.Dir(), "jdk", selector))
	}
}

func linkLatest() error {
	files, _ := readDir(filepath.Join(cfg.Dir(), "jdk"))
	if err := discovery.DeleteCacheFile(cfg.Dir()); err != nil {
		log.Warn("Failed to delete cache file: ", err)
	}
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
						log.Info(sourceVersion + " -/> " + target)
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
			log.Info(v.String() + " -> " + target)
			os.Remove(source)
			if err := os.Symlink(target, source); err != nil {
				return err
			}
		}
	}
	return linkAlias("default", jdks)
}

func linkAliasName(name string) error {
	var jdks, err = Ls(false)
	if err != nil {
		return err
	}
	return linkAlias(name, jdks)
}

func linkAlias(name string, jdks []discovery.JDK) error {
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
			log.Info(sourceRef + " -> " + target)
			os.Remove(source)
			if err := os.Symlink(target, source); err != nil {
				return err
			}
		}
	} else {
		err := os.Remove(source)
		if err == nil {
			log.Info(sourceRef + " -/> " + sourceTarget)
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
