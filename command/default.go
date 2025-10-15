package command

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/felipebz/javm/cfg"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// NewDefaultCommand creates the `javm default` CLI command.
func NewDefaultCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "default [version]",
		Short: "Set the default Java version to use in new shells",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return pflag.ErrHelp
			}
			ver := args[0]
			if err := SetDefaultVersion(ver); err != nil {
				log.Fatal(err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Default Java version set to %s\n", ver)
			return nil
		},
	}
}

// SetDefaultVersion writes the provided selector to the default-version file
// in the javm configuration directory. It creates the directory if needed.
func SetDefaultVersion(selector string) error {
	dir := cfg.Dir()
	if err := os.MkdirAll(dir, 0o777); err != nil {
		return err
	}
	file := filepath.Join(dir, "default-version")
	return os.WriteFile(file, []byte(selector), 0o666)
}
