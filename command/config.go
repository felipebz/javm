package command

import (
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/felipebz/javm/cfg"
	"github.com/spf13/cobra"
)

func NewConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage javm configuration",
	}

	getCmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Get effective value for a config key",
		Run:   runGetConfig,
	}

	setCmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a config key to a value",
		Run:   runSetConfig,
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all effective configuration keys and values",
		Run:   runListConfig,
	}

	unsetCmd := &cobra.Command{
		Use:   "unset <key>",
		Short: "Remove a key from the user configuration (revert to default)",
		Run:   runUnsetConfig,
	}

	cmd.AddCommand(getCmd, setCmd, unsetCmd, listCmd)
	return cmd
}

func runGetConfig(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		_ = cmd.Usage()
		os.Exit(0)
	}
	key := args[0]
	if !cfg.IsKnownKey(key) {
		fmt.Fprintf(os.Stderr, "error: unknown key \"%s\"\n", key)
		os.Exit(2)
	}
	v, err := cfg.EffectiveValue(key)
	if err != nil {
		handleConfigError(err)
	}
	fmt.Println(v)
}

func runSetConfig(cmd *cobra.Command, args []string) {
	if len(args) != 2 {
		_ = cmd.Usage()
		os.Exit(0)
	}
	key := args[0]
	val := args[1]
	if !cfg.IsKnownKey(key) {
		fmt.Fprintf(os.Stderr, "error: unknown key \"%s\"\n", key)
		os.Exit(2)
	}
	if err := cfg.SetValue(key, val); err != nil {
		if errors.Is(err, cfg.ErrInvalidConfigFile) {
			fmt.Fprintf(os.Stderr, "error: invalid config file; please fix or remove: %s\n", cfg.ConfigFile())
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "error: failed to write config: %v\n", err)
		os.Exit(3)
	}
}

func runListConfig(cmd *cobra.Command, args []string) {
	lines, err := cfg.ListEffective()
	if err != nil {
		handleConfigError(err)
	}
	sort.Strings(lines)
	for _, l := range lines {
		fmt.Println(l)
	}
}

func runUnsetConfig(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		_ = cmd.Usage()
		os.Exit(0)
	}
	key := args[0]
	if !cfg.IsKnownKey(key) {
		fmt.Fprintf(os.Stderr, "error: unknown key \"%s\"\n", key)
		os.Exit(2)
	}
	if err := cfg.UnsetValue(key); err != nil {
		if errors.Is(err, cfg.ErrInvalidConfigFile) {
			fmt.Fprintf(os.Stderr, "error: invalid config file; please fix or remove: %s\n", cfg.ConfigFile())
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "error: failed to write config: %v\n", err)
		os.Exit(3)
	}
}

func handleConfigError(err error) {
	if errors.Is(err, cfg.ErrInvalidConfigFile) {
		fmt.Fprintf(os.Stderr, "error: invalid config file; please fix or remove: %s\n", cfg.ConfigFile())
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}
