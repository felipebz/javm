package command

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

// The sentinels below are the command-layer error classes consumed by the
// process entry point when it chooses an exit status.
var (
	ErrUsage    = errors.New("usage error")
	ErrNotFound = errors.New("not found")
	ErrNetwork  = errors.New("network error")
)

// UsageError marks an error caused by invalid user input or command usage.
func UsageError(err error) error {
	if err == nil || errors.Is(err, ErrUsage) {
		return err
	}
	return fmt.Errorf("%w: %w", ErrUsage, err)
}

// NotFoundError marks an error caused by a requested resource not existing.
func NotFoundError(err error) error {
	if err == nil || errors.Is(err, ErrNotFound) {
		return err
	}
	return fmt.Errorf("%w: %w", ErrNotFound, err)
}

// NetworkError marks an error caused by a remote service or download.
func NetworkError(err error) error {
	if err == nil || errors.Is(err, ErrNetwork) {
		return err
	}
	return fmt.Errorf("%w: %w", ErrNetwork, err)
}

// UsageArgs adapts a Cobra positional-argument validator to the command error
// contract used by javm.
func UsageArgs(validate cobra.PositionalArgs) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if err := validate(cmd, args); err != nil {
			return UsageError(err)
		}
		return nil
	}
}
