package command

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"

	"github.com/felipebz/javm/cfg"
	"github.com/felipebz/javm/discovery"
	"github.com/felipebz/javm/javaversion"
	"github.com/spf13/cobra"
)

func NewUninstallCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall [version to uninstall]",
		Short: "Uninstall JDK",
		Args:  UsageArgs(cobra.ExactArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			if isSystemSelector(args[0]) {
				return UsageError(fmt.Errorf("link to system JDK can only be removed with 'unlink' (e.g. 'javm unlink %s')", sanitizeTerminalText(args[0])))
			}
			pins := resolvePins(cmd.Context())
			removed, err := uninstall(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if err := warnLinkFailures(cmd.Context(), linkLatest(cmd.Context())); err != nil {
				return err
			}
			if err := warnLinkFailures(cmd.Context(), refreshAliasLinks(cmd.Context())); err != nil {
				return err
			}
			warnUnresolvedPins(cmd.Context(), pins, removed)
			return nil
		},
		Example: "  javm uninstall temurin@21.0.4",
	}
}

func uninstall(ctx context.Context, selector string) (string, error) {
	jdks, err := LsContext(ctx, true)
	if err == nil {
		var identifier string
		if identifier, err = uniqueManagedMatch(jdks, selector); err == nil {
			return identifier, removeAndReport(ctx, identifier)
		}
	}
	// A JDK whose metadata cannot be read is not listed, so it can only be
	// removed by its exact directory name.
	if errors.Is(err, ErrNotFound) || errors.Is(err, ErrUsage) {
		if dst, dstErr := managedInstallDestination(cfg.Dir(), selector); dstErr == nil {
			if info, statErr := os.Lstat(dst); statErr == nil && info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
				return selector, removeAndReport(ctx, selector)
			}
		}
	}
	if errors.Is(err, ErrNotFound) {
		if all, lsErr := LsContext(ctx, false); lsErr == nil {
			jdk, findErr := FindBestMatchJDK(all, selector)
			switch {
			case findErr != nil:
			case jdk.Source == "javm" && isSystemSelector(jdk.Identifier):
				return "", UsageError(fmt.Errorf("%s is a link to a JDK outside javm; remove it with 'javm unlink %s'",
					sanitizeTerminalText(jdk.Identifier), sanitizeTerminalText(jdk.Identifier)))
			case jdk.Source != "javm":
				return "", NotFoundError(fmt.Errorf("%s is not managed by javm (%s); remove it with the tool that installed it",
					sanitizeTerminalText(jdk.Identifier), sanitizeTerminalText(jdk.Path)))
			}
		}
	}
	return "", err
}

func uniqueManagedMatch(jdks []discovery.JDK, selector string) (string, error) {
	for _, jdk := range jdks {
		if jdk.Identifier == selector {
			return jdk.Identifier, nil
		}
	}
	rng, err := javaversion.ParseRange(selector)
	if err != nil {
		return "", UsageError(err)
	}
	matches := filterByRange(jdks, rng)
	switch len(matches) {
	case 0:
		_, err := FindBestMatchJDK(jdks, selector)
		return "", err
	case 1:
		return matches[0].Identifier, nil
	}
	sort.Slice(matches, func(i, j int) bool { return newerJDKFirst(matches[i], matches[j]) })
	identifiers := make([]string, len(matches))
	for i, jdk := range matches {
		identifiers[i] = sanitizeTerminalText(jdk.Identifier)
	}
	return "", UsageError(fmt.Errorf("%s matches %d installed JDKs: %s; pass the exact identifier",
		sanitizeTerminalText(selector), len(matches), strings.Join(identifiers, ", ")))
}

func removeAndReport(ctx context.Context, identifier string) error {
	if err := removeManagedJDK(identifier); err != nil {
		return err
	}
	loggerFromContext(ctx).Infof("Removed %s", sanitizeTerminalText(identifier))
	return nil
}

type resolvedPin struct {
	aliasSelector
	identifier string
}

func resolvePins(ctx context.Context) []resolvedPin {
	pinned, err := pinnedSelectors(ctx)
	var jdks []discovery.JDK
	if err == nil {
		jdks, err = LsContext(ctx, false)
	}
	if err != nil {
		warnOnce(ctx, fmt.Errorf("cannot check the default version and aliases: %w", err))
		return nil
	}
	var pins []resolvedPin
	for _, p := range pinned {
		if p.err != nil {
			continue
		}
		if jdk, err := resolveJDKFromList(jdks, p.selector); err == nil {
			pins = append(pins, resolvedPin{aliasSelector: p, identifier: jdk.Identifier})
		}
	}
	return pins
}

// Runs after the removal: only the new list shows whether another JDK took over.
func warnUnresolvedPins(ctx context.Context, pins []resolvedPin, removed string) {
	pins = slices.DeleteFunc(pins, func(p resolvedPin) bool { return p.identifier != removed })
	if len(pins) == 0 {
		return
	}
	jdks, err := LsContext(ctx, false)
	if err != nil {
		warnOnce(ctx, fmt.Errorf("cannot check the default version and aliases: %w", err))
		return
	}
	for _, p := range pins {
		if _, err := resolveJDKFromList(jdks, p.selector); err == nil {
			continue
		}
		selector := sanitizeTerminalText(p.selector)
		if p.name == "" {
			warnOnce(ctx, fmt.Errorf("default version %s no longer matches an installed JDK; set another with 'javm default <version>'", selector))
			continue
		}
		name := sanitizeTerminalText(p.name)
		warnOnce(ctx, fmt.Errorf("alias %s (%s) no longer matches an installed JDK; change it with 'javm alias %s <version>' or remove it with 'javm unalias %s'",
			name, selector, name, name))
	}
}

// removeManagedJDK deletes jdk/<identifier> only when it is a real directory,
// never a symlink created by link or linkLatest.
func removeManagedJDK(identifier string) error {
	dst, err := managedInstallDestination(cfg.Dir(), identifier)
	if err != nil {
		return err
	}
	info, err := os.Lstat(dst)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%q is not a managed JDK directory", identifier)
	}
	return os.RemoveAll(dst)
}
