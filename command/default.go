package command

import (
	"errors"
	"fmt"
	"io/fs"
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
		Use:   "default [selector]",
		Short: "Show or set the default JDK for new shells",
		Long:  "Without a selector, print the default JDK selector. With one, set the default JDK for new shells.",
		Args:  UsageArgs(cobra.MaximumNArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return printDefault(cmd)
			}
			ver := strings.TrimSpace(args[0])
			if err := validateDefaultSelector(args[0]); err != nil {
				return err
			}
			// Resolve as `javm use` would, so new shells do not start with an
			// error. The selector is stored as typed, so 21 follows updates.
			jdk, err := resolveJDKContext(cmd.Context(), ver)
			if err != nil {
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
			if err := os.Remove(filepath.Join(cfg.Dir(), "default.alias")); err != nil && !errors.Is(err, fs.ErrNotExist) {
				warnOnce(cmd.Context(), fmt.Errorf("remove legacy default.alias: %w", err))
			}
			// default-version is already written and selection does not use the link.
			return warnLinkFailures(cmd.Context(), pointLink(cmd.Context(), "default", jdk.Path))
		},
	}
}

// printDefault writes the stored default selector without resolving it, so
// it needs neither discovery nor a writable JAVM_HOME.
func printDefault(cmd *cobra.Command) error {
	selector, err := effectiveDefaultVersion(func(err error) { warnOnce(cmd.Context(), err) })
	if errors.Is(err, fs.ErrNotExist) {
		return NotFoundError(errors.New("no default JDK is set; set one with 'javm default <selector>'"))
	}
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), sanitizeTerminalText(selector)); err != nil {
		return fmt.Errorf("write default version: %w", err)
	}
	return nil
}

// effectiveDefaultVersion returns default-version, or the legacy default.alias
// when default-version does not exist, and passes the migration hints to warn.
// Shell startup reads only default-version, as javm 0.17 did.
func effectiveDefaultVersion(warn func(error)) (string, error) {
	selector, err := readDefaultVersion()
	legacy, legacyErr := readAlias("default")
	hasLegacy := !errors.Is(legacyErr, fs.ErrNotExist)
	switch {
	case !hasLegacy:
		return selector, err
	case !errors.Is(err, fs.ErrNotExist):
		warn(errors.New("ignoring legacy default.alias because default-version is set; remove it with 'javm unalias default'"))
		return selector, err
	case legacyErr != nil:
		return "", legacyErr
	}
	warn(errors.New("using legacy default.alias as the default; run 'javm default <selector>' to migrate"))
	return legacy, validateDefaultSelector(legacy)
}

func effectiveDefaultPath() string {
	if _, err := os.Lstat(defaultVersionPath()); errors.Is(err, fs.ErrNotExist) {
		return filepath.Join(cfg.Dir(), "default.alias")
	}
	return defaultVersionPath()
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
