package command

import (
	"errors"
	"fmt"
	"sort"

	"github.com/felipebz/javm/cfg"
	"github.com/spf13/cobra"
)

func NewConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "config",
		Short:        "Manage javm configuration",
		SilenceUsage: true,
	}

	getCmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Get effective value for a config key",
		Args:  cobra.ExactArgs(1),
		RunE:  runGetConfig,
	}

	setCmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a config key to a value",
		Args:  cobra.ExactArgs(2),
		RunE:  runSetConfig,
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all effective configuration keys and values",
		Args:  cobra.NoArgs,
		RunE:  runListConfig,
	}

	unsetCmd := &cobra.Command{
		Use:   "unset <key>",
		Short: "Remove a key from the user configuration (revert to default)",
		Args:  cobra.ExactArgs(1),
		RunE:  runUnsetConfig,
	}

	cmd.AddCommand(getCmd, setCmd, unsetCmd, listCmd)
	return cmd
}

func runGetConfig(cmd *cobra.Command, args []string) error {
	key := args[0]
	if !cfg.IsKnownKey(key) {
		return fmt.Errorf("unknown key %q", key)
	}
	v, err := cfg.EffectiveValue(key)
	if err != nil {
		return configError(err)
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), v)
	if err != nil {
		return fmt.Errorf("write config value: %w", err)
	}
	return nil
}

func runSetConfig(cmd *cobra.Command, args []string) error {
	key := args[0]
	val := args[1]
	if !cfg.IsKnownKey(key) {
		return fmt.Errorf("unknown key %q", key)
	}
	if err := cfg.SetValue(key, val); err != nil {
		return fmt.Errorf("failed to write config: %w", configError(err))
	}
	return nil
}

func runListConfig(cmd *cobra.Command, args []string) error {
	lines, err := cfg.ListEffective()
	if err != nil {
		return configError(err)
	}
	sort.Strings(lines)
	for _, l := range lines {
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), l); err != nil {
			return fmt.Errorf("write config list: %w", err)
		}
	}
	return nil
}

func runUnsetConfig(cmd *cobra.Command, args []string) error {
	key := args[0]
	if !cfg.IsKnownKey(key) {
		return fmt.Errorf("unknown key %q", key)
	}
	if err := cfg.UnsetValue(key); err != nil {
		return fmt.Errorf("failed to remove config value: %w", configError(err))
	}
	return nil
}

func configError(err error) error {
	if errors.Is(err, cfg.ErrInvalidConfigFile) {
		return fmt.Errorf("invalid config file; please fix or remove %s: %w", cfg.ConfigFile(), err)
	}
	return err
}
