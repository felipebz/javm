package command

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/felipebz/javm/cfg"
	"github.com/felipebz/javm/semver"
	"github.com/spf13/cobra"
)

// NewDefaultCommand creates the `javm default` CLI command.
func NewDefaultCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "default [version]",
		Short: "Set the default Java version to use in new shells",
		Args:  cobra.ExactArgs(1),
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
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create configuration directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".default-version-*")
	if err != nil {
		return fmt.Errorf("create temporary default version: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("restrict temporary default version permissions: %w", err)
	}
	if _, err := tmp.WriteString(selector); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write default version: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync default version: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close default version: %w", err)
	}
	if err := os.Rename(tmpPath, filepath.Join(dir, "default-version")); err != nil {
		return fmt.Errorf("replace default version: %w", err)
	}
	return nil
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
		return fmt.Errorf("default version selector contains a forbidden control character")
	}
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return fmt.Errorf("default version selector cannot be empty")
	}
	if _, err := semver.ParseRange(selector); err != nil {
		return fmt.Errorf("invalid default version selector %q: %w", selector, err)
	}
	return nil
}
