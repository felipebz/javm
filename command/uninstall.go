package command

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/felipebz/javm/cfg"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func NewUninstallCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall [version to uninstall]",
		Short: "Uninstall JDK",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return pflag.ErrHelp
			}
			if strings.HasPrefix(args[0], "system@") {
				return fmt.Errorf("Link to system JDK can only be removed with 'unlink' (e.g. 'javm unlink %s')", args[0])
			}
			if err := uninstall(args[0]); err != nil {
				return err
			}
			if err := LinkLatest(); err != nil {
				return err
			}
			return nil
		},
		Example: "  javm uninstall 1.8",
	}
}

func uninstall(selector string) error {
	ver, err := LsBestMatch(selector, true)
	if err != nil {
		return err
	}
	return os.RemoveAll(filepath.Join(cfg.Dir(), "jdk", ver))
}
