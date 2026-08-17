package command

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/felipebz/javm/cfg"
	"github.com/felipebz/javm/internal/state"
	"github.com/spf13/cobra"
)

func NewAliasCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "alias [name] [version]",
		Short: "Resolve or update an alias",
		Args:  UsageArgs(cobra.RangeArgs(1, 2)),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if len(args) == 1 {
				value, err := readAlias(name)
				if err != nil && !os.IsNotExist(err) {
					return err
				}
				if value != "" {
					if _, err := fmt.Fprintln(cmd.OutOrStdout(), value); err != nil {
						return fmt.Errorf("write alias: %w", err)
					}
				}
				return nil
			}
			if err := setAlias(name, args[1]); err != nil {
				return err
			}
			if err := linkAliasName(cmd.Context(), name); err != nil {
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
		Args:  UsageArgs(cobra.ExactArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := setAlias(args[0], ""); err != nil {
				return err
			}
			if err := linkAliasName(cmd.Context(), args[0]); err != nil {
				return err
			}
			return nil
		},
	}
}

func validateAliasName(name string) error {
	if name == "" || name == "." || name == ".." {
		return UsageError(fmt.Errorf("invalid alias name %q", name))
	}
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-' {
			continue
		}
		return UsageError(fmt.Errorf("invalid alias name %q: only letters, numbers, '.', '_' and '-' are allowed", name))
	}
	return nil
}

func aliasPath(name string) (string, error) {
	if err := validateAliasName(name); err != nil {
		return "", err
	}

	root := filepath.Clean(cfg.Dir())
	path := filepath.Join(root, name+".alias")
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return "", fmt.Errorf("resolve alias path: %w", err)
	}
	if rel == "." || rel == ".." || filepath.IsAbs(rel) ||
		strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("alias path escapes configuration directory")
	}
	return path, nil
}

func setAlias(name string, ver string) error {
	path, err := aliasPath(name)
	if err != nil {
		return err
	}
	if ver == "" {
		return state.WithFileLock(path, func() error {
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("remove alias %q: %w", name, err)
			}
			return nil
		})
	}
	return state.WithFileLock(path, func() error {
		if err := state.AtomicWriteFile(path, []byte(ver), 0o600); err != nil {
			return fmt.Errorf("write alias %q: %w", name, err)
		}
		return nil
	})
}

func readAlias(name string) (string, error) {
	path, err := aliasPath(name)
	if err != nil {
		return "", err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read alias %q: %w", name, err)
	}
	return string(b), nil
}

func getAlias(name string) string {
	value, _ := readAlias(name)
	return value
}
