package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sort"
	"strings"

	"github.com/felipebz/javm/command"
	"github.com/felipebz/javm/discoapi"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/term"
)

var version string
var commit string
var date string
var rootCmd *cobra.Command

type simpleFormatter struct{}

func (f *simpleFormatter) Format(entry *log.Entry) ([]byte, error) {
	b := &bytes.Buffer{}
	fmt.Fprintf(b, "%s ", entry.Message)
	keys := make([]string, 0, len(entry.Data))
	for key := range entry.Data {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := entry.Data[k]
		fmt.Fprintf(b, "%s=%+v ", k, v)
	}
	b.WriteByte('\n')
	return b.Bytes(), nil
}

type application struct {
	out      io.Writer
	err      io.Writer
	logger   *log.Logger
	client   *discoapi.Client
	terminal func(io.Writer) bool
}

func newRootCommand(app application) *cobra.Command {
	root := &cobra.Command{
		Use:          "javm",
		Long:         "Java Version Manager (https://javm.dev).",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if showVersion, _ := cmd.Flags().GetBool("version"); !showVersion {
				return pflag.ErrHelp
			}
			msg := version
			details := make([]string, 0, 2)
			if commit != "" {
				details = append(details, "commit "+commit)
			}
			if date != "" {
				details = append(details, "built at "+date)
			}
			if len(details) > 0 {
				msg = fmt.Sprintf("%s (%s)", version, strings.Join(details, ", "))
			}
			if _, err := fmt.Fprintln(cmd.OutOrStdout(), msg); err != nil {
				return fmt.Errorf("write version: %w", err)
			}
			return nil
		},
	}
	root.SetOut(app.out)
	root.SetErr(app.err)
	root.AddCommand(
		command.NewInstallCommand(app.client),
		command.NewUninstallCommand(),
		command.NewLinkCommand(),
		command.NewUnlinkCommand(),
		command.NewUseCommand(),
		command.NewCurrentCommand(),
		command.NewLsCommand(),
		command.NewLsRemoteCommand(app.client),
		command.NewDeactivateCommand(),
		command.NewAliasCommand(),
		command.NewUnaliasCommand(),
		command.NewLsDistributionsCommand(app.client),
		command.NewWhichCommand(),
		command.NewInitCommand(),
		command.NewDiscoverCommand(),
		command.NewDefaultCommand(),
		command.NewConfigCommand(),
	)
	root.Flags().Bool("version", false, "version of javm")
	root.PersistentFlags().Bool("debug", false, "enable verbose debug logging")
	root.PersistentFlags().Bool("quiet", false, "suppress non-error logs")
	root.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		level := log.InfoLevel
		if dbg, _ := cmd.Flags().GetBool("debug"); dbg {
			level = log.DebugLevel
		} else if q, _ := cmd.Flags().GetBool("quiet"); q {
			level = log.WarnLevel
		}
		app.logger.SetLevel(level)
		app.logger.SetOutput(cmd.ErrOrStderr())
		quiet, _ := cmd.Flags().GetBool("quiet")
		showProgress := false
		if app.terminal != nil {
			showProgress = !quiet && app.terminal(cmd.ErrOrStderr())
		}
		ctx := command.WithRuntime(cmd.Context(), command.Runtime{
			Logger:       app.logger,
			Err:          cmd.ErrOrStderr(),
			ShowProgress: showProgress,
		})
		cmd.SetContext(ctx)
	}
	return root
}

type fileDescriptor interface {
	Fd() uintptr
}

func isTerminalWriter(w io.Writer) bool {
	fd, ok := w.(fileDescriptor)
	return ok && term.IsTerminal(int(fd.Fd()))
}

func main() {
	logger := log.New()
	logger.SetFormatter(&simpleFormatter{})
	logger.SetLevel(log.InfoLevel)
	client := discoapi.NewClient()
	client.Logger = logger
	rootCmd = newRootCommand(application{out: os.Stdout, err: os.Stderr, logger: logger, client: client, terminal: isTerminalWriter})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	err := rootCmd.ExecuteContext(ctx)
	stop()
	if err != nil {
		os.Exit(-1)
	}
}
