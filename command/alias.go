package command

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/felipebz/javm/cfg"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func NewAliasCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "alias [name] [version]",
		Short: "Resolve or update an alias",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return pflag.ErrHelp
			}
			name := args[0]
			if len(args) == 1 {
				if value := getAlias(name); value != "" {
					fmt.Println(value)
				}
				return nil
			}
			if err := setAlias(name, args[1]); err != nil {
				return err
			}
			if err := linkAliasName(name); err != nil {
				return err
			}
			return nil
		},
		Example: "  javm alias default 1.8\n" +
			"  javm alias default # show value bound to an alias",
	}
}

func NewUnaliasCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "unalias [name]",
		Short: "Delete an alias",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return pflag.ErrHelp
			}
			if err := setAlias(args[0], ""); err != nil {
				return err
			}
			return nil
		},
	}
}

func setAlias(name string, ver string) (err error) {
	if ver == "" {
		err = os.Remove(filepath.Join(cfg.Dir(), name+".alias"))
	} else {
		err = os.WriteFile(filepath.Join(cfg.Dir(), name+".alias"), []byte(ver), 0644)
	}
	return
}

func getAlias(name string) string {
	b, err := os.ReadFile(filepath.Join(cfg.Dir(), name+".alias"))
	if err != nil {
		return ""
	}
	return string(b)
}
