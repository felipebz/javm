package command

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"

	"github.com/felipebz/javm/cfg"
	"github.com/felipebz/javm/discovery"
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
	ver, err := LsBestMatchContext(ctx, selector, true)
	if err == nil {
		return ver, removeManagedJDK(ver)
	}
	// A JDK whose metadata cannot be read is not listed, so it can only be
	// removed by its exact directory name.
	if errors.Is(err, ErrNotFound) || errors.Is(err, ErrUsage) {
		if dst, dstErr := managedInstallDestination(cfg.Dir(), selector); dstErr == nil {
			if info, statErr := os.Lstat(dst); statErr == nil && info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
				return selector, removeManagedJDK(selector)
			}
		}
	}
	return "", err
}

type resolvedPin struct {
	aliasSelector
	identifier string
}

func resolvePins(ctx context.Context) []resolvedPin {
	pinned, err := pinnedSelectors()
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
