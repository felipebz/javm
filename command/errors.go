package command

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// The sentinels below are the command-layer error classes consumed by the
// process entry point when it chooses an exit status.
var (
	ErrUsage            = errors.New("usage error")
	ErrShellIntegration = errors.New("shell integration is not active")
	ErrNotFound         = errors.New("not found")
	ErrNetwork          = errors.New("network error")
)

// UsageError marks an error caused by invalid user input or command usage.
func UsageError(err error) error {
	if err == nil || errors.Is(err, ErrUsage) {
		return err
	}
	return fmt.Errorf("%w: %w", ErrUsage, err)
}

func usageCause(err error) error {
	for e := err; e != nil; e = errors.Unwrap(e) {
		if multi, ok := e.(interface{ Unwrap() []error }); ok {
			if errs := multi.Unwrap(); len(errs) == 2 && errs[0] == ErrUsage {
				return errs[1]
			}
		}
	}
	return err
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

// Without a RunE, Cobra shows the help before validating Args.
func showHelp(*cobra.Command, []string) error {
	return pflag.ErrHelp
}
