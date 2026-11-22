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

const linkUsage = "usage: javm link <path> [--name <version>]"

func NewLinkCommand() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "link <path> [--name <version>]",
		Short: "Register a JDK installed outside javm",
		Long: "Create JAVM_HOME/jdk/system@<version> for a JDK installed outside javm, so it can be selected as\n" +
			"system@<version>. The version is read from the JDK; --name chooses another one.",
		Args: UsageArgs(cobra.MaximumNArgs(2)),
		RunE: func(cmd *cobra.Command, args []string) error {
			var dir string
			switch {
			case len(args) == 0:
				return UsageError(errors.New("a JDK path is required; " + linkUsage))
			case len(args) == 2 && isSystemSelector(args[0]) && name == "":
				name, dir = args[0], args[1]
				version := sanitizeTerminalText(strings.TrimPrefix(name, "system@"))
				warnOnce(cmd.Context(), fmt.Errorf("'javm link system@%s <path>' is deprecated; use 'javm link <path> --name %s'", version, version))
			case len(args) == 2:
				return UsageError(errors.New("too many arguments; " + linkUsage))
			default:
				dir = args[0]
			}
			if info, err := os.Stat(dir); isSystemSelector(dir) && (err != nil || !info.IsDir()) {
				return UsageError(fmt.Errorf("%q is not a JDK directory; to see the JDK behind a link, use 'javm which %s'", dir, sanitizeTerminalText(dir)))
			}
			return linkJDK(cmd.Context(), dir, name)
		},
		Example: "  javm link /Library/Java/JavaVirtualMachines/temurin-21.jdk\n" +
			"  javm link /opt/jdk-21 --name 21-custom",
	}
	cmd.Flags().StringVar(&name, "name", "", "version for the system@<version> identifier instead of the JDK's own")
	return cmd
}

func NewUnlinkCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "unlink <identifier|version|path>",
		Short: "Remove a link created by 'javm link'",
		Args:  UsageArgs(cobra.ExactArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			return unlinkJDK(cmd.Context(), args[0])
		},
		Example: "  javm unlink system@21.0.4\n" +
			"  javm unlink 21.0.4\n" +
			"  javm unlink /Library/Java/JavaVirtualMachines/temurin-21.jdk",
	}
}

// linkJDK creates jdk/system@<version> for the JDK at dir. Linking the same
// path again is a no-op; a different path under the same name is refused.
func linkJDK(ctx context.Context, dir, name string) error {
	if err := assertJavaDistribution(dir, runtime.GOOS); err != nil {
		return err
	}
	version, err := linkVersion(ctx, dir, name)
	if err != nil {
		return err
	}
	identifier := "system@" + version
	source := filepath.Join(cfg.Dir(), "jdk", identifier)
	info, err := os.Lstat(source)
	switch {
	case err == nil && info.Mode()&os.ModeSymlink != 0 && canonicalHome(source) == canonicalHome(dir):
		loggerFromContext(ctx).Info(sanitizeTerminalText(identifier) + " already links to " + sanitizeTerminalText(dir))
		return nil
	case err == nil:
		existing := source
		if target, readErr := os.Readlink(source); readErr == nil {
			existing = target
		}
		return UsageError(fmt.Errorf("%s already links to %s; choose another name with --name or remove it with 'javm unlink %s'",
			sanitizeTerminalText(identifier), sanitizeTerminalText(existing), sanitizeTerminalText(identifier)))
	case !errors.Is(err, fs.ErrNotExist):
		return err
	}
	if jdks, err := LsContext(ctx, false); err == nil {
		for _, jdk := range jdks {
			if jdk.Identifier != identifier && canonicalHome(jdk.Path) == canonicalHome(dir) {
				loggerFromContext(ctx).Infof("%s is already found by discovery as %s; selecting it does not need a link",
					sanitizeTerminalText(dir), sanitizeTerminalText(jdk.Identifier))
				break
			}
		}
	}
	if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
		return err
	}
	if err := createSymlink(dir, source); err != nil {
		return withSymlinkHint(&symlinkError{action: "create symlink", link: source, target: dir, err: err})
	}
	loggerFromContext(ctx).Info(sanitizeTerminalText(identifier) + " -> " + sanitizeTerminalText(dir))
	return invalidateDiscoveryCache()
}

// linkVersion returns the version part of the system@ identifier: name when
// given, otherwise the JDK's own version read as discovery reads it, without
// the build as in managed directory names. Java 8 and earlier report 1.x
// versions (1.8.0_402), which javm writes without the "1." (8.0.402).
func linkVersion(ctx context.Context, dir, name string) (string, error) {
	if name != "" {
		name = strings.TrimPrefix(name, "system@")
		if _, err := javaversion.ParseVersion("system@" + name); err != nil {
			return "", UsageError(err)
		}
		return name, nil
	}
	unreadable := UsageError(fmt.Errorf("cannot read the version of the JDK at %s; choose one with --name", sanitizeTerminalText(dir)))
	absolute, err := filepath.Abs(dir)
	if err != nil {
		return "", unreadable
	}
	parent := filepath.Dir(absolute)
	jdk, ok, err := discovery.ValidateJDKContext(ctx, os.DirFS(parent), discovery.ExecRunner{}, parent, filepath.Base(absolute), "link")
	if err != nil || !ok {
		return "", unreadable
	}
	version := jdk.Version
	if rest, legacy := strings.CutPrefix(version, "1."); legacy {
		version = strings.ReplaceAll(rest, "_", ".")
	}
	parsed, err := javaversion.ParseVersion(version)
	if err != nil {
		return "", unreadable
	}
	return parsed.WithoutBuild(), nil
}

// unlinkJDK removes a jdk/system@* symlink chosen by identifier, bare
// version or target path. Managed JDK directories are never removed.
func unlinkJDK(ctx context.Context, arg string) error {
	jdkDir := filepath.Join(cfg.Dir(), "jdk")
	identifier := arg
	if !isSystemSelector(identifier) {
		identifier = "system@" + arg
	}
	if _, err := javaversion.ParseVersion(identifier); err == nil && isSymlink(filepath.Join(jdkDir, identifier)) {
		return removeSystemLink(ctx, identifier)
	}
	if info, err := os.Stat(arg); err == nil && info.IsDir() {
		entries, err := os.ReadDir(jdkDir)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		var matches []string
		for _, entry := range entries {
			path := filepath.Join(jdkDir, entry.Name())
			if isSystemSelector(entry.Name()) && isSymlink(path) && canonicalHome(path) == canonicalHome(arg) {
				matches = append(matches, entry.Name())
			}
		}
		switch len(matches) {
		case 0:
		case 1:
			return removeSystemLink(ctx, matches[0])
		default:
			for i := range matches {
				matches[i] = sanitizeTerminalText(matches[i])
			}
			return UsageError(fmt.Errorf("%s is linked as %s; unlink one of them by name", sanitizeTerminalText(arg), strings.Join(matches, ", ")))
		}
	}
	if dst, err := managedInstallDestination(cfg.Dir(), arg); err == nil {
		if info, err := os.Lstat(dst); err == nil && info.IsDir() {
			return UsageError(fmt.Errorf("%s is a JDK installed by javm, not a link; remove it with 'javm uninstall %s'",
				sanitizeTerminalText(arg), sanitizeTerminalText(arg)))
		}
	}
	return NotFoundError(fmt.Errorf("no link matches %q; see 'javm ls' for the available JDKs", arg))
}

func isSymlink(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode()&os.ModeSymlink != 0
}

func removeSystemLink(ctx context.Context, identifier string) error {
	if err := os.Remove(filepath.Join(cfg.Dir(), "jdk", identifier)); err != nil {
		return err
	}
	loggerFromContext(ctx).Info("Removed link " + sanitizeTerminalText(identifier))
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
	if err := linkDefault(ctx, jdks); err != nil && !refresh.add(err) {
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
		source := filepath.Join(cfg.Dir(), "jdk", name)
		oldTarget, _ := os.Readlink(source)
		err := os.Remove(source)
		if err == nil {
			loggerFromContext(ctx).Info(sanitizeTerminalText(name) + " -/> " + sanitizeTerminalText(oldTarget))
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
	link := linkAlias
	if name == "default" {
		link = func(ctx context.Context, _ string, jdks []discovery.JDK) error { return linkDefault(ctx, jdks) }
	}
	if err := link(ctx, name, jdks); err != nil {
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

// linkDefault points jdk/default at the JDK the default resolves to, as
// `use --default` resolves it. Managed JDKs are tried first because the
// discovery cache may not list one that was just installed.
func linkDefault(ctx context.Context, managed []discovery.JDK) error {
	selector, err := effectiveDefaultVersion(func(err error) { warnOnce(ctx, err) })
	if err != nil && !errors.Is(err, fs.ErrNotExist) && !errors.Is(err, ErrUsage) {
		warnOnce(ctx, err)
		return nil
	}
	if err == nil {
		jdk, findErr := FindBestMatchJDK(managed, selector)
		if findErr != nil {
			all, lsErr := LsContext(ctx, false)
			if lsErr != nil {
				return lsErr
			}
			jdk, findErr = FindBestMatchJDK(all, selector)
		}
		if findErr == nil {
			return pointLink(ctx, "default", jdk.Path)
		}
	}
	return removeStaleAliasLink(ctx, "default", filepath.Join(cfg.Dir(), "jdk", "default"))
}

// linkAlias points jdk/<name> at the JDK the alias selects in jdks, using the
// JDK's own path so discovered JDKs outside JAVM_HOME work too. When the alias
// is unset or matches nothing in jdks, a stale jdk/<name> symlink is removed;
// the alias file itself is left alone.
func linkAlias(ctx context.Context, name string, jdks []discovery.JDK) error {
	if err := validateAliasName(name); err != nil {
		return err
	}
	if value := getAlias(name); value != "" {
		if jdk, err := FindBestMatchJDK(jdks, value); err == nil {
			return pointLink(ctx, name, jdk.Path)
		}
	}
	return removeStaleAliasLink(ctx, name, filepath.Join(cfg.Dir(), "jdk", name))
}

func pointLink(ctx context.Context, name, target string) error {
	if getLink(name) == canonicalHome(target) {
		return nil
	}
	source := filepath.Join(cfg.Dir(), "jdk", name)
	if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
		return fmt.Errorf("create managed JDK directory: %w", err)
	}
	loggerFromContext(ctx).Info(name + " -> " + sanitizeTerminalText(target))
	return replaceSymlink(target, source)
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
	target, _ := os.Readlink(source)
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
