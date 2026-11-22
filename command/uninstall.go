package command

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/felipebz/javm/cfg"
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
			if err := uninstall(cmd.Context(), args[0]); err != nil {
				return err
			}
			if err := warnLinkFailures(cmd.Context(), linkLatest(cmd.Context())); err != nil {
				return err
			}
			return nil
		},
		Example: "  javm uninstall temurin@21.0.4",
	}
}

func uninstall(ctx context.Context, selector string) error {
	ver, err := LsBestMatchContext(ctx, selector, true)
	if err == nil {
		return removeManagedJDK(ver)
	}
	// A JDK whose metadata cannot be read is not listed, so it can only be
	// removed by its exact directory name.
	if errors.Is(err, ErrNotFound) || errors.Is(err, ErrUsage) {
		if dst, dstErr := managedInstallDestination(cfg.Dir(), selector); dstErr == nil {
			if info, statErr := os.Lstat(dst); statErr == nil && info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
				return removeManagedJDK(selector)
			}
		}
	}
	return err
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
