package command

import (
	"context"
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
				return UsageError(fmt.Errorf("Link to system JDK can only be removed with 'unlink' (e.g. 'javm unlink %s')", args[0]))
			}
			if err := uninstall(cmd.Context(), args[0]); err != nil {
				return err
			}
			if err := linkLatest(cmd.Context()); err != nil {
				return err
			}
			return nil
		},
		Example: "  javm uninstall 1.8",
	}
}

func uninstall(ctx context.Context, selector string) error {
	ver, err := LsBestMatchContext(ctx, selector, true)
	if err != nil {
		return err
	}
	return removeManagedJDK(ver)
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
