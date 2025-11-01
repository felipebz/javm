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

	deactivateCmd := &cobra.Command{
		Use:   "deactivate",
		Short: "Undo effects of `javm` on current shell",
		RunE: func(cmd *cobra.Command, args []string) error {
			out, err := command.Deactivate()
			if err != nil {
				return err
			}
			fd3, _ := cmd.Flags().GetString("fd3")
			printForShellToEval(out, fd3)
			return nil
		},
	}
	deactivateCmd.Flags().String("fd3", "", "")
	deactivateCmd.Flags().MarkHidden("fd3")
	rootCmd.AddCommand(
		command.NewInstallCommand(client),
		&cobra.Command{
			Use:   "uninstall [version to uninstall]",
			Short: "Uninstall JDK",
			RunE: func(cmd *cobra.Command, args []string) error {
				if len(args) == 0 {
					return pflag.ErrHelp
				}
				if strings.HasPrefix(args[0], "system@") {
					return fmt.Errorf("Link to system JDK can only be removed with 'unlink' (e.g. 'javm unlink %s')", args[0])
				}
				if err := command.Uninstall(args[0]); err != nil {
					return err
				}
				if err := command.LinkLatest(); err != nil {
					return err
				}
				return nil
			},
			Example: "  javm uninstall 1.8",
		},
		&cobra.Command{
			Use:   "link [name] [path]",
			Short: "Resolve or update a link",
			RunE: func(cmd *cobra.Command, args []string) error {
				if len(args) == 0 {
					if err := command.LinkLatest(); err != nil {
						return err
					}
					return nil
				}
				if len(args) == 1 {
					if value := command.GetLink(args[0]); value != "" {
						fmt.Println(value)
					}
				} else if err := command.Link(args[0], args[1]); err != nil {
					return err
				}
				return nil
			},
			Example: "  javm link system@1.8.20 /Library/Java/JavaVirtualMachines/jdk1.8.0_20.jdk\n" +
				"  javm link system@1.8.20 # show link target",
		},
		&cobra.Command{
			Use:   "unlink [name]",
			Short: "Delete a link",
			RunE: func(cmd *cobra.Command, args []string) error {
				if len(args) == 0 {
					return pflag.ErrHelp
				}
				if err := command.Link(args[0], ""); err != nil {
					return err
				}
				return nil
			},
			Example: "  javm unlink system@1.8.20",
		},
		useCmd,
		&cobra.Command{
			Use:   "current",
			Short: "Display currently 'use'ed version",
			Run: func(cmd *cobra.Command, args []string) {
				ver := command.Current()
				if ver != "" {
					fmt.Println(ver)
				}
			},
		},
		command.NewLsCommand(),
		command.NewLsRemoteCommand(client),
		deactivateCmd,
		&cobra.Command{
			Use:   "alias [name] [version]",
			Short: "Resolve or update an alias",
			RunE: func(cmd *cobra.Command, args []string) error {
				if len(args) == 0 {
					return pflag.ErrHelp
				}
				name := args[0]
				if len(args) == 1 {
					if value := command.GetAlias(name); value != "" {
						fmt.Println(value)
					}
					return nil
				}
				if err := command.SetAlias(name, args[1]); err != nil {
					return err
				}
				if err := command.LinkAlias(name); err != nil {
					return err
				}
				return nil
			},
			Example: "  javm alias default 1.8\n" +
				"  javm alias default # show value bound to an alias",
		},
		&cobra.Command{
			Use:   "unalias [name]",
			Short: "Delete an alias",
			RunE: func(cmd *cobra.Command, args []string) error {
				if len(args) == 0 {
					return pflag.ErrHelp
				}
				if err := command.SetAlias(args[0], ""); err != nil {
					return err
				}
				return nil
			},
		},
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
	printForShellToEval(out, fd3)
	return nil
}

func printForShellToEval(out []string, fd3 string) {
	if fd3 != "" {
		os.WriteFile(fd3, []byte(strings.Join(out, "\n")), 0666)
	} else {
		fd := os.NewFile(3, "fd3")
		for _, line := range out {
			fmt.Fprintln(fd, line)
		}
	}
}
