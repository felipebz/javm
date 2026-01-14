package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/felipebz/javm/cfg"
	"github.com/felipebz/javm/command"
	"github.com/felipebz/javm/discoapi"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var version string
var commit string
var date string
var rootCmd *cobra.Command

func init() {
	log.SetFormatter(&simpleFormatter{})
	log.SetLevel(log.InfoLevel)
}

type simpleFormatter struct{}

func (f *simpleFormatter) Format(entry *log.Entry) ([]byte, error) {
	b := &bytes.Buffer{}
	fmt.Fprintf(b, "%s ", entry.Message)
	for k, v := range entry.Data {
		fmt.Fprintf(b, "%s=%+v ", k, v)
	}
	b.WriteByte('\n')
	return b.Bytes(), nil
}

func main() {
	rootCmd = &cobra.Command{
		Use:  "javm",
		Long: "Java Version Manager (https://javm.dev).",
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
			fmt.Println(msg)
			return nil
		},
	}
	client := discoapi.NewClient()

	useCmd := &cobra.Command{
		Use:   "use [version to use]",
		Short: "Modify PATH & JAVA_HOME to use specific JDK",
		RunE: func(cmd *cobra.Command, args []string) error {
			var ver string
			if len(args) == 0 {
				ver = cfg.ReadJavaVersion()
				if ver == "" {
					return pflag.ErrHelp
				}
			} else {
				ver = args[0]
			}
			fd3, _ := cmd.Flags().GetString("fd3")
			return use(ver, fd3)
		},
		Example: "  javm use 1.8\n" +
			"  javm use ~1.8.73 # same as \">=1.8.73 <1.9.0\"",
	}
	useCmd.Flags().String("fd3", "", "")
	useCmd.Flags().MarkHidden("fd3")

	rootCmd.AddCommand(
		command.NewInstallCommand(client),
		command.NewUninstallCommand(),
		command.NewLinkCommand(),
		command.NewUnlinkCommand(),
		useCmd,
		command.NewCurrentCommand(),
		command.NewLsCommand(),
		command.NewLsRemoteCommand(client),
		command.NewDeactivateCommand(),
		command.NewAliasCommand(),
		command.NewUnaliasCommand(),
		command.NewLsDistributionsCommand(client),
		command.NewWhichCommand(),
		command.NewInitCommand(),
		command.NewDiscoverCommand(),
		command.NewDefaultCommand(),
		command.NewConfigCommand(),
	)
	rootCmd.Flags().Bool("version", false, "version of javm")
	rootCmd.PersistentFlags().Bool("debug", false, "enable verbose debug logging")
	rootCmd.PersistentFlags().Bool("quiet", false, "suppress non-error logs")
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if dbg, _ := cmd.Flags().GetBool("debug"); dbg {
			log.SetLevel(log.DebugLevel)
		} else if q, _ := cmd.Flags().GetBool("quiet"); q {
			log.SetLevel(log.WarnLevel)
		}
	}

	if err := rootCmd.Execute(); err != nil {
		os.Exit(-1)
	}
}

func use(ver string, fd3 string) error {
	out, err := command.Use(ver)
	if err != nil {
		return err
	}
	command.PrintForShellToEval(out, fd3)
	return nil
}
