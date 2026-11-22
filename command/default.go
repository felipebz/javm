package command

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/felipebz/javm/cfg"
	"github.com/felipebz/javm/internal/state"
	"github.com/felipebz/javm/javaversion"
	"github.com/spf13/cobra"
)

// NewDefaultCommand creates the `javm default` CLI command.
func NewDefaultCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "default [version]",
		Short: "Set the default Java version to use in new shells",
		Args:  UsageArgs(cobra.ExactArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			ver := strings.TrimSpace(args[0])
			if err := validateDefaultSelector(args[0]); err != nil {
				return err
			}
			// Resolve as `javm use` would, so new shells do not start with an
			// error. The selector is stored as typed, so 21 follows updates.
			if _, err := resolveJDKContext(cmd.Context(), ver); err != nil {
				if errors.Is(err, ErrNotFound) {
					return NotFoundError(fmt.Errorf("no installed JDK matches %q; install it first with 'javm install %s'",
						sanitizeTerminalText(ver), sanitizeTerminalText(ver)))
				}
				return err
			}
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

func defaultVersionPath() string {
	return filepath.Join(cfg.Dir(), "default-version")
}

func readDefaultVersion() (string, error) {
	data, err := os.ReadFile(defaultVersionPath())
	if err != nil {
		return "", err
	}
	selector, err := persistedSelector(data)
	if err != nil {
		err = UsageError(fmt.Errorf("default version selector %w", err))
	} else {
		err = validateDefaultSelector(selector)
	}
	if err != nil {
		return "", fmt.Errorf("invalid persisted default version: %w", err)
	}
	return selector, nil
}

func persistedSelector(data []byte) (string, error) {
	selector := strings.TrimRight(string(data), " \t\r\n")
	if strings.ContainsAny(selector, "\r\n\x00") {
		return "", errors.New("contains a forbidden control character")
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
	if _, err := javaversion.ParseRange(selector); err != nil {
		return UsageError(fmt.Errorf("invalid default version selector %q: %w", selector, err))
	}
	return nil
}
