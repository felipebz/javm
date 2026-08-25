package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/felipebz/javm/command"
	"github.com/felipebz/javm/discoapi"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func TestExitCodeMapsStableErrorClasses(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "success", want: exitSuccess},
		{name: "generic failure", err: errors.New("failure"), want: exitFailure},
		{name: "usage", err: command.UsageError(errors.New("bad arguments")), want: exitUsage},
		{name: "inactive shell integration", err: fmt.Errorf("context: %w", command.ErrShellIntegration), want: exitUsage},
		{name: "not found", err: command.NotFoundError(errors.New("missing JDK")), want: exitNotFound},
		{name: "network", err: command.NetworkError(errors.New("API unavailable")), want: exitNetwork},
		{name: "discoapi network", err: fmt.Errorf("request failed: %w", discoapi.ErrNetwork), want: exitNetwork},
		{name: "timeout", err: context.DeadlineExceeded, want: exitTimeout},
		{name: "interrupted", err: context.Canceled, want: exitInterrupted},
		{name: "child process", err: fakeExitError{code: 7}, want: 7},
		{name: "help", err: pflag.ErrHelp, want: exitSuccess},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := exitCode(tt.err); got != tt.want {
				t.Fatalf("exitCode(%v) = %d, want %d", tt.err, got, tt.want)
			}
		})
	}
}

type fakeExitError struct{ code int }

func (e fakeExitError) Error() string { return "child exited" }
func (e fakeExitError) ExitCode() int { return e.code }

func TestRootCommandRejectsUnexpectedArguments(t *testing.T) {
	logger := log.New()
	root := newRootCommand(application{
		out:    &bytes.Buffer{},
		err:    &bytes.Buffer{},
		logger: logger,
		client: discoapi.NewClient(),
	})
	root.SetArgs([]string{"current", "extra"})

	err := root.Execute()
	if !errors.Is(err, command.ErrUsage) {
		t.Fatalf("root.Execute() error = %v, want ErrUsage", err)
	}
	if got := exitCode(err); got != exitUsage {
		t.Fatalf("exitCode(%v) = %d, want %d", err, got, exitUsage)
	}
}

func TestSimpleFormatterSortsFields(t *testing.T) {
	formatter := &simpleFormatter{}
	formatted, err := formatter.Format(&log.Entry{
		Message: "message",
		Data:    log.Fields{"zeta": 2, "alpha": 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(formatted), "message alpha=1 zeta=2 \n"; got != want {
		t.Fatalf("formatted log = %q, want %q", got, want)
	}
}

func TestRootCommandRoutesOutputAndConfiguresRuntime(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		wantLevel      log.Level
		wantProgress   bool
		wantDiagnostic bool
	}{
		{name: "default", wantLevel: log.InfoLevel, wantProgress: true, wantDiagnostic: true},
		{name: "quiet", args: []string{"--quiet"}, wantLevel: log.WarnLevel},
		{name: "debug", args: []string{"--debug"}, wantLevel: log.DebugLevel, wantProgress: true, wantDiagnostic: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			logger := log.New()
			logger.SetFormatter(&simpleFormatter{})
			client := discoapi.NewClient()
			client.Logger = logger
			root := newRootCommand(application{
				out:      &stdout,
				err:      &stderr,
				logger:   logger,
				client:   client,
				terminal: func(io.Writer) bool { return true },
			})
			root.AddCommand(&cobra.Command{
				Use: "runtime-test",
				RunE: func(cmd *cobra.Command, _ []string) error {
					runtime := command.RuntimeFromContext(cmd.Context())
					if runtime.Logger != logger {
						return fmt.Errorf("runtime logger was not injected")
					}
					if runtime.Err != &stderr {
						return fmt.Errorf("runtime stderr was not injected")
					}
					if runtime.ShowProgress != tt.wantProgress {
						return fmt.Errorf("ShowProgress = %v, want %v", runtime.ShowProgress, tt.wantProgress)
					}
					logger.Info("diagnostic")
					_, err := fmt.Fprintln(cmd.OutOrStdout(), "data")
					return err
				},
			})
			root.SetArgs(append(tt.args, "runtime-test"))

			if err := root.Execute(); err != nil {
				t.Fatal(err)
			}
			if logger.GetLevel() != tt.wantLevel {
				t.Fatalf("logger level = %s, want %s", logger.GetLevel(), tt.wantLevel)
			}
			if got := stdout.String(); got != "data\n" {
				t.Fatalf("stdout = %q, want data only", got)
			}
			if got := stderr.String(); (got != "") != tt.wantDiagnostic {
				t.Fatalf("stderr = %q, diagnostic expected = %v", got, tt.wantDiagnostic)
			}
		})
	}
}
