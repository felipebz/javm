package command

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/felipebz/javm/cfg"
	"github.com/felipebz/javm/discovery"
	"github.com/felipebz/javm/javaversion"
	"github.com/spf13/cobra"
)

type updateOptions struct {
	keep   bool
	dryRun bool
}

func NewUpdateCommand(client PackagesWithInfoClient) *cobra.Command {
	var opts updateOptions
	cmd := &cobra.Command{
		Use:   "update [version or alias]",
		Short: "Update installed JDKs to the latest matching build",
		Args:  UsageArgs(cobra.MaximumNArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			var selector string
			if len(args) == 1 {
				selector = args[0]
			}
			err := runUpdate(cmd.Context(), client, cmd.OutOrStdout(), selector, opts)
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				cmd.SilenceUsage = true
			}
			return err
		},
		Example: "  javm update zulu@25\n" +
			"  javm update        # every installed distribution@feature line\n" +
			"  javm update --dry-run",
	}
	cmd.Flags().BoolVar(&opts.keep, "keep", false, "keep the previously selected JDK after updating")
	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "show available updates without installing")
	return cmd
}

func runUpdate(ctx context.Context, client PackagesWithInfoClient, out io.Writer, selector string, opts updateOptions) error {
	managed, err := LsContext(ctx, true)
	if err != nil {
		return err
	}

	explicit := selector != ""
	var targets []string
	if explicit {
		if aliasValue := getAlias(selector); aliasValue != "" {
			selector = aliasValue
		}
		targets = []string{selector}
	} else {
		targets = updateLines(managed)
		if len(targets) == 0 {
			if _, err := fmt.Fprintln(out, "no managed JDKs to update"); err != nil {
				return fmt.Errorf("write update status: %w", err)
			}
			return nil
		}
	}

	var errs []error
	changedAny := false
	for _, target := range targets {
		if err := ctx.Err(); err != nil {
			errs = append(errs, err)
			break
		}
		changed, err := updateOne(ctx, client, out, target, managed, opts)
		changedAny = changedAny || changed
		if err == nil {
			continue
		}
		// Stop at the first explicit or context error, but still relink below
		// when the failure came after a JDK was installed.
		if explicit || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			errs = append(errs, err)
			break
		}
		if errors.Is(err, ErrNotFound) {
			loggerFromContext(ctx).Warn(err)
			continue
		}
		errs = append(errs, err)
	}

	if changedAny && !opts.dryRun {
		if err := linkLatest(ctx); err != nil {
			errs = append(errs, err)
		} else if err := refreshAliasLinks(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// updateLines lists each installed managed distribution@feature line once.
func updateLines(managed []discovery.JDK) []string {
	var lines []string
	for _, jdk := range managed {
		v, err := javaversion.ParseVersion(jdk.Identifier)
		if err != nil || v.Qualifier() == "" {
			continue
		}
		line := fmt.Sprintf("%s@%d", v.Qualifier(), v.Feature())
		if !slices.Contains(lines, line) {
			lines = append(lines, line)
		}
	}
	slices.Sort(lines)
	return lines
}

func updateOne(ctx context.Context, client PackagesWithInfoClient, out io.Writer, target string, managed []discovery.JDK, opts updateOptions) (bool, error) {
	current, err := FindBestMatchJDK(managed, target)
	if err != nil {
		return false, err
	}
	rng, err := javaversion.ParseRange(target)
	if err != nil {
		return false, UsageError(err)
	}
	installed := installedVersion(current)
	if installed == nil {
		return false, fmt.Errorf("cannot parse installed JDK %q", current.Identifier)
	}
	// A selector without distribution updates the installed distribution,
	// not java.default_distribution.
	if rng.Qualifier == "" || rng.Qualifier == "*" {
		rng.Qualifier = installed.Qualifier()
	}

	idx, err := makePackageIndex(ctx, client, runtime.GOOS, runtime.GOARCH, rng.Qualifier)
	if err != nil {
		return false, err
	}
	latest := idx.latestMatching(rng)
	if latest == nil {
		return false, NotFoundError(fmt.Errorf("no remote package matches %s", target))
	}
	if !isNewerPackage(idx, latest, installed) {
		if _, err := fmt.Fprintf(out, "%s: up to date (%s)\n", sanitizeTerminalText(target), sanitizeTerminalText(installed.String())); err != nil {
			return false, fmt.Errorf("write update status: %w", err)
		}
		return false, nil
	}
	if _, err := fmt.Fprintf(out, "%s: %s -> %s\n", sanitizeTerminalText(target), sanitizeTerminalText(installed.String()), sanitizeTerminalText(latest.String())); err != nil {
		return false, fmt.Errorf("write update status: %w", err)
	}
	if opts.dryRun {
		return false, nil
	}

	info, err := client.GetPackageInfoContext(ctx, idx.ByVersion[latest].Id)
	if err != nil {
		return false, NetworkError(err)
	}
	dst, err := managedInstallDestination(cfg.Dir(), latest.WithoutBuild())
	if err != nil {
		return false, err
	}
	replace := false
	if _, err := os.Lstat(dst); err == nil {
		replace = true
	} else if !os.IsNotExist(err) {
		return false, fmt.Errorf("inspect installation destination: %w", err)
	}
	if err := fetchAndInstall(ctx, latest, info, dst, replace); err != nil {
		return false, err
	}

	if filepath.Base(dst) == current.Identifier || opts.keep {
		return true, nil
	}
	after, err := LsContext(ctx, true)
	if err != nil {
		return true, err
	}
	holders, err := selectorsHolding(after, current.Identifier)
	if err != nil {
		return true, err
	}
	if len(holders) > 0 {
		loggerFromContext(ctx).Infof("Keeping %s: still selected by %s", current.Identifier, strings.Join(holders, ", "))
		return true, nil
	}
	if err := removeManagedJDK(current.Identifier); err != nil {
		return true, err
	}
	loggerFromContext(ctx).Infof("Removed %s", current.Identifier)
	return true, nil
}

// installedVersion names the package a managed JDK came from. The directory
// name follows the DiscoAPI scheme; only the build comes from the release
// file, whose version can differ in format (Temurin 8 reports 1.8.0_502).
func installedVersion(jdk discovery.JDK) *javaversion.Version {
	id, err := javaversion.ParseVersion(jdk.Identifier)
	if err != nil {
		return nil
	}
	v, err := javaversion.ParseVersion(id.Qualifier() + "@" + jdk.Version)
	if err != nil || v.TrimTo(javaversion.VPJavaPatch) != id.TrimTo(javaversion.VPJavaPatch) {
		return id
	}
	build, ok := v.BuildNumber()
	if !ok {
		return id
	}
	withBuild, err := javaversion.ParseVersion(fmt.Sprintf("%s+%d", id.String(), build))
	if err != nil {
		return id
	}
	return withBuild
}

// isNewerPackage reports whether latest supersedes installed. When installed is
// still listed remotely, the index order decides, so a Zulu rebuild with a lower
// build but newer distribution_version counts as newer.
func isNewerPackage(idx *packageIndex, latest, installed *javaversion.Version) bool {
	installedBuild, installedHasBuild := installed.BuildNumber()
	if installedHasBuild {
		for _, v := range idx.Sorted {
			build, ok := v.BuildNumber()
			if ok && build == installedBuild && v.TrimTo(javaversion.VPJavaPatch) == installed.TrimTo(javaversion.VPJavaPatch) {
				return idx.compare(latest, v) > 0
			}
		}
	}
	lr, lrErr := javaversion.ParseVersion(latest.TrimTo(javaversion.VPJavaPatch))
	ir, irErr := javaversion.ParseVersion(installed.TrimTo(javaversion.VPJavaPatch))
	if lrErr == nil && irErr == nil {
		if c := lr.Compare(ir); c != 0 {
			return c > 0
		}
	}
	// Same release: an installation without a build number counts as current.
	latestBuild, latestHasBuild := latest.BuildNumber()
	return latestHasBuild && installedHasBuild && latestBuild > installedBuild
}

type aliasSelector struct {
	label    string
	selector string
}

// selectorsHolding lists default-version and aliases that still resolve to
// identifier, so update keeps a JDK the user pinned.
func selectorsHolding(jdks []discovery.JDK, identifier string) ([]string, error) {
	var pinned []aliasSelector
	defaultVersion, err := readDefaultVersion()
	switch {
	case err == nil:
		pinned = append(pinned, aliasSelector{label: "default-version", selector: defaultVersion})
	case !errors.Is(err, os.ErrNotExist):
		return nil, err
	}
	names, err := aliasNames()
	if err != nil {
		return nil, err
	}
	for _, name := range names {
		value, err := readAlias(name)
		if err != nil {
			return nil, err
		}
		pinned = append(pinned, aliasSelector{label: "alias " + name, selector: value})
	}

	var holders []string
	for _, p := range pinned {
		if jdk, err := resolveJDKFromList(jdks, p.selector); err == nil && jdk.Identifier == identifier {
			holders = append(holders, p.label)
		}
	}
	return holders, nil
}

// refreshAliasLinks repoints jdk/<alias> symlinks after JDKs were replaced;
// linkLatest already handles the default alias.
func refreshAliasLinks(ctx context.Context) error {
	names, err := aliasNames()
	if err != nil {
		return err
	}
	names = slices.DeleteFunc(names, func(name string) bool { return name == "default" })
	if len(names) == 0 {
		return nil
	}
	// Named aliases may point at discovered JDKs, as linkAliasName links them.
	jdks, err := LsContext(ctx, false)
	if err != nil {
		return err
	}
	for _, name := range names {
		if err := linkAlias(ctx, name, jdks); err != nil {
			return err
		}
	}
	return invalidateDiscoveryCache()
}

// aliasNames lists the valid alias names stored in cfg.Dir().
func aliasNames() ([]string, error) {
	entries, err := os.ReadDir(cfg.Dir())
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read alias directory: %w", err)
	}
	var names []string
	for _, entry := range entries {
		name, ok := strings.CutSuffix(entry.Name(), ".alias")
		if !ok || entry.IsDir() || validateAliasName(name) != nil {
			continue
		}
		names = append(names, name)
	}
	return names, nil
}
