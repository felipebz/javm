package command

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/felipebz/javm/cfg"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func NewWhichCommand() *cobra.Command {
	var whichHome bool
	cmd := &cobra.Command{
		Use:   "which [version]",
		Short: "Display path to installed JDK",
		RunE: func(cmd *cobra.Command, args []string) error {
			var ver string
			if len(args) == 0 {
				ver = cfg.ReadJavaVersion()
				if ver == "" {
					return pflag.ErrHelp
				}
			} else {
				ver = args[0]
			}
			dir, _ := Which(ver, whichHome)
			if dir != "" {
				fmt.Fprintln(cmd.OutOrStdout(), dir)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&whichHome, "home", false, "Account for platform differences so that value could be used as JAVA_HOME (e.g. append \"/Contents/Home\" on macOS)")
	return cmd
}

func Which(selector string, home bool) (string, error) {
	aliasValue := GetAlias(selector)
	if aliasValue != "" {
		selector = aliasValue
	}
	ver, err := LsBestMatch(selector)
	if err != nil {
		return "", err
	}
	path := filepath.Join(cfg.Dir(), "jdk", ver)
	if home && runtime.GOOS == "darwin" {
		path = filepath.Join(path, "Contents", "Home")
	}
	return path, nil
}
