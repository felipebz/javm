package command

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/felipebz/javm/cfg"
	"github.com/felipebz/javm/internal/state"
	"github.com/felipebz/javm/semver"
	"github.com/spf13/cobra"
)

// NewDefaultCommand creates the `javm default` CLI command.
func NewDefaultCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "default [version]",
		Short: "Set the default Java version to use in new shells",
		Args:  UsageArgs(cobra.ExactArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			ver := args[0]
			if err := SetDefaultVersion(ver); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Default Java version set to %s\n", ver); err != nil {
				return fmt.Errorf("write default version confirmation: %w", err)
			}
			return nil
		},
	}
}

// SetDefaultVersion writes the provided selector to the default-version file
// in the javm configuration directory. It creates the directory if needed.
func SetDefaultVersion(selector string) error {
	if err := validateDefaultSelector(selector); err != nil {
		return err
	}
	selector = strings.TrimSpace(selector)

	dir := cfg.Dir()
	path := filepath.Join(dir, "default-version")
	return state.WithFileLock(path, func() error {
		if err := state.AtomicWriteFile(path, []byte(selector), 0o600); err != nil {
			return fmt.Errorf("write default version: %w", err)
		}
		return nil
	})
}

func readDefaultVersion() (string, error) {
	data, err := os.ReadFile(filepath.Join(cfg.Dir(), "default-version"))
	if err != nil {
		return "", err
	}
	selector := string(data)
	if err := validateDefaultSelector(selector); err != nil {
		return "", fmt.Errorf("invalid persisted default version: %w", err)
	}
	return strings.TrimSpace(selector), nil
}

func validateDefaultSelector(selector string) error {
	if strings.ContainsAny(selector, "\r\n\x00") {
		return UsageError(fmt.Errorf("default version selector contains a forbidden control character"))
	}
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return UsageError(fmt.Errorf("default version selector cannot be empty"))
	}
	if _, err := semver.ParseRange(selector); err != nil {
		return UsageError(fmt.Errorf("invalid default version selector %q: %w", selector, err))
	}
	return nil
}
