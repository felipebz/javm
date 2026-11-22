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
	"syscall"

	"github.com/felipebz/javm/cfg"
	"github.com/felipebz/javm/discovery"
	"github.com/felipebz/javm/javaversion"
	"github.com/spf13/cobra"
)

var removePath = os.Remove

// createSymlink and hostGOOS are variables so tests can simulate Windows
// refusing to create symlinks.
var (
	createSymlink = os.Symlink
	hostGOOS      = runtime.GOOS
)

// errPrivilegeNotHeld is ERROR_PRIVILEGE_NOT_HELD (1314), returned by
// os.Symlink on Windows without Developer Mode or an elevated process.
// syscall.Errno exists on every platform, so no build-tagged file is needed.
const errPrivilegeNotHeld = syscall.Errno(1314)

// symlinkHint explains a symlink failure caused by missing Windows
// privileges, or returns "" for any other error.
func symlinkHint(err error) string {
	if err == nil || hostGOOS != "windows" || !errors.Is(err, errPrivilegeNotHeld) {
		return ""
	}
	return "creating symlinks on Windows requires Developer Mode or an elevated prompt"
}

// withSymlinkHint appends symlinkHint to err when it applies.
func withSymlinkHint(err error) error {
	if hint := symlinkHint(err); hint != "" {
		return fmt.Errorf("%w (%s)", err, hint)
	}
	return err
}

// symlinkError names the link and its target once with native separators.
// *os.LinkError and *fs.PathError repeat the paths in their text, so only their
// cause is printed; the original error stays reachable through Unwrap.
type symlinkError struct {
	action, link, target string
	err                  error
}

func (e *symlinkError) Error() string {
	cause := e.err
	var linkErr *os.LinkError
	var pathErr *fs.PathError
	switch {
	case errors.As(cause, &linkErr):
		cause = linkErr.Err
	case errors.As(cause, &pathErr):
		cause = pathErr.Err
	}
	message := e.action + " " + sanitizeTerminalText(e.link)
	if e.target != "" {
		message += " -> " + sanitizeTerminalText(e.target)
	}
	return message + ": " + cause.Error()
}

func (e *symlinkError) Unwrap() error { return e.err }

// linkRefreshError collects failures to create or replace convenience links
// (jdk/<dist@feature.interim>, jdk/default, jdk/<alias>). Selection does not
// depend on these links, so callers that only refresh them may warn instead.
type linkRefreshError struct{ failures []error }

func (e *linkRefreshError) Error() string   { return errors.Join(e.failures...).Error() }
func (e *linkRefreshError) Unwrap() []error { return e.failures }

// add records err when it is a link failure and reports whether it was one.
func (e *linkRefreshError) add(err error) bool {
	var failure *linkRefreshError
	if !errors.As(err, &failure) {
		return false
	}
	e.failures = append(e.failures, failure.failures...)
	return true
}

func (e *linkRefreshError) result() error {
	if len(e.failures) == 0 {
		return nil
	}
	return e
}

// warnLinkFailures logs each convenience-link failure in err once per command
// and drops it; any other error is returned unchanged.
func warnLinkFailures(ctx context.Context, err error) error {
	var failure *linkRefreshError
	if err == nil || !errors.As(err, &failure) || failure != err {
		return err
	}
	for _, linkErr := range failure.failures {
		if hint := symlinkHint(linkErr); hint != "" {
			linkErr = fmt.Errorf("%w (%s; JDK selection is not affected)", linkErr, hint)
		}
		warnOnce(ctx, linkErr)
	}
	return nil
}

func replaceSymlink(target, source string) error {
	err := removePath(source)
	switch {
	case err == nil:
	case errors.Is(err, fs.ErrNotExist):
	default:
		return &linkRefreshError{failures: []error{&symlinkError{action: "remove existing symlink", link: source, err: err}}}
	}
	if err := createSymlink(target, source); err != nil {
		return &linkRefreshError{failures: []error{&symlinkError{action: "create symlink", link: source, target: target, err: err}}}
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
				return withSymlinkHint(linkLatest(cmd.Context()))
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
		Example: "  javm link system@21 /Library/Java/JavaVirtualMachines/temurin-21.jdk\n" +
			"  javm link system@21 # show link target",
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
		Example: "  javm unlink system@21",
	}
}

func link(ctx context.Context, selector string, dir string) error {
	if !isSystemSelector(selector) {
		return UsageError(errors.New("name must begin with 'system@' (e.g. 'system@21')"))
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
		source := filepath.Join(cfg.Dir(), "jdk", selector)
		if err := createSymlink(dir, source); err != nil {
			return withSymlinkHint(&symlinkError{action: "create symlink", link: source, target: dir, err: err})
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
	var refresh linkRefreshError
	for _, v := range javaversion.VersionSlice(managedVersions(jdks)).TrimTo(javaversion.VPInterim) {
		name := v.TrimTo(javaversion.VPInterim)
		target := filepath.Join(cfg.Dir(), "jdk", v.String())
		if v.Prerelease() != "" || links[name] == target || isSystemSelector(name) {
			continue
		}
		loggerFromContext(ctx).Info(v.String() + " -> " + target)
		// One link that cannot be created must not keep the others stale.
		if err := replaceSymlink(target, filepath.Join(cfg.Dir(), "jdk", name)); err != nil && !refresh.add(err) {
			return err
		}
	}
	aliasJDKs, err := defaultAliasJDKs(ctx, jdks)
	if err != nil {
		return err
	}
	if err := linkAlias(ctx, "default", aliasJDKs); err != nil && !refresh.add(err) {
		return err
	}
	return refresh.result()
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
		return withSymlinkHint(err)
	}
	return invalidateDiscoveryCache()
}

func invalidateDiscoveryCache() error {
	if err := discovery.DeleteCacheFile(cfg.Dir()); err != nil {
		return fmt.Errorf("invalidate discovery cache: %w", err)
	}
	return nil
}

// defaultAliasJDKs returns the list to resolve the default alias against: the
// managed JDKs when they match it, otherwise the full discovery, so a default
// that selects a discovered JDK keeps its link. Managed JDKs are tried first
// because the discovery cache may not list one that was just installed.
func defaultAliasJDKs(ctx context.Context, managed []discovery.JDK) ([]discovery.JDK, error) {
	value := getAlias("default")
	if value == "" {
		return managed, nil
	}
	if _, err := FindBestMatchJDK(managed, value); err == nil {
		return managed, nil
	}
	return LsContext(ctx, false)
}

// linkAlias points jdk/<name> at the JDK the alias selects in jdks, using the
// JDK's own path so discovered JDKs outside JAVM_HOME work too. When the alias
// is unset or matches nothing in jdks, a stale jdk/<name> symlink is removed;
// the alias file itself is left alone.
func linkAlias(ctx context.Context, name string, jdks []discovery.JDK) error {
	if err := validateAliasName(name); err != nil {
		return err
	}
	source := filepath.Join(cfg.Dir(), "jdk", name)
	if value := getAlias(name); value != "" {
		if jdk, err := FindBestMatchJDK(jdks, value); err == nil {
			if getLink(name) == canonicalHome(jdk.Path) {
				return nil
			}
			if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
				return fmt.Errorf("create managed JDK directory: %w", err)
			}
			loggerFromContext(ctx).Info(name + " -> " + sanitizeTerminalText(jdk.Path))
			return replaceSymlink(jdk.Path, source)
		}
	}
	return removeStaleAliasLink(ctx, name, source)
}

// removeStaleAliasLink removes jdk/<name> when it is a symlink; a real
// directory with that name is never touched.
func removeStaleAliasLink(ctx context.Context, name, source string) error {
	info, err := os.Lstat(source)
	if errors.Is(err, fs.ErrNotExist) || (err == nil && info.Mode()&os.ModeSymlink == 0) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect %s: %w", sanitizeTerminalText(source), err)
	}
	target := getLink(name)
	if err := os.Remove(source); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	loggerFromContext(ctx).Info(name + " -/> " + sanitizeTerminalText(target))
	return nil
}

func getLink(name string) string {
	res, err := filepath.EvalSymlinks(filepath.Join(cfg.Dir(), "jdk", name))
	if err != nil {
		return ""
	}
	return res
}
