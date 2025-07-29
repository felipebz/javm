package main

import (
	"bytes"
	"fmt"
	"github.com/felipebz/javm/command"
	"github.com/felipebz/javm/discoapi"
	"github.com/felipebz/javm/semver"
	"github.com/goccy/go-yaml"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"os"
	"strings"
)

var version string
var rootCmd *cobra.Command

func init() {
	log.SetFormatter(&simpleFormatter{})
	// todo: make it configurable through the command line
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
		Long: "Java Version Manager (https://github.com/felipebz/javm).",
		RunE: func(cmd *cobra.Command, args []string) error {
			if showVersion, _ := cmd.Flags().GetBool("version"); !showVersion {
				return pflag.ErrHelp
			}
			fmt.Println(version)
			return nil
		},
	}
	var whichHome bool
	whichCmd := &cobra.Command{
		Use:   "which [version]",
		Short: "Display path to installed JDK",
		RunE: func(cmd *cobra.Command, args []string) error {
			var ver string
			if len(args) == 0 {
				ver = rc().JDK
				if ver == "" {
					return pflag.ErrHelp
				}
			} else {
				ver = args[0]
			}
			dir, _ := command.Which(ver, whichHome)
			if dir != "" {
				fmt.Println(dir)
			}
			return nil
		},
	}
	whichCmd.Flags().BoolVar(&whichHome, "home", false,
		"Account for platform differences so that value could be used as JAVA_HOME (e.g. append \"/Contents/Home\" on macOS)")
	var trimTo string
	lsCmd := &cobra.Command{
		Use:   "ls",
		Short: "List installed versions",
		RunE: func(cmd *cobra.Command, args []string) error {
			var r *semver.Range
			if len(args) > 0 {
				var err error
				r, err = semver.ParseRange(args[0])
				if err != nil {
					log.Fatal(err)
				}
			}
			vs, err := command.Ls()
			if err != nil {
				log.Fatal(err)
			}
			if trimTo != "" {
				vs = semver.VersionSlice(vs).TrimTo(parseTrimTo(trimTo))
			}
			for _, v := range vs {
				if r != nil && !r.Contains(v) {
					continue
				}
				fmt.Println(v)
			}
			return nil
		},
	}
	for _, cmd := range []*cobra.Command{lsCmd} {
		cmd.Flags().StringVar(&trimTo, "latest", "",
			"Part of the version to trim to (\"major\", \"minor\" or \"patch\")")
	}
	client := discoapi.NewClient()
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
					log.Fatal("Link to system JDK can only be removed with 'unlink'" +
						" (e.g. 'javm unlink " + args[0] + "')")
				}
				err := command.Uninstall(args[0])
				if err != nil {
					log.Fatal(err)
				}
				if err := command.LinkLatest(); err != nil {
					log.Fatal(err)
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
						log.Fatal(err)
					}
					return nil
				}
				if len(args) == 1 {
					if value := command.GetLink(args[0]); value != "" {
						fmt.Println(value)
					}
				} else if err := command.Link(args[0], args[1]); err != nil {
					log.Fatal(err)
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
					log.Fatal(err)
				}
				return nil
			},
			Example: "  javm unlink system@1.8.20",
		},
		&cobra.Command{
			Use:   "use [version to use]",
			Short: "Modify PATH & JAVA_HOME to use specific JDK",
			RunE: func(cmd *cobra.Command, args []string) error {
				var ver string
				if len(args) == 0 {
					ver = rc().JDK
					if ver == "" {
						return pflag.ErrHelp
					}
				} else {
					ver = args[0]
				}
				return use(ver)
			},
			Example: "  javm use 1.8\n" +
				"  javm use ~1.8.73 # same as \">=1.8.73 <1.9.0\"",
		},
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
		lsCmd,
		command.NewLsRemoteCommand(client),
		&cobra.Command{
			Use:   "deactivate",
			Short: "Undo effects of `javm` on current shell",
			RunE: func(cmd *cobra.Command, args []string) error {
				out, err := command.Deactivate()
				if err != nil {
					log.Fatal(err)
				}
				printForShellToEval(out)
				return nil
			},
		},
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
					log.Fatal(err)
				}
				if err := command.LinkAlias(name); err != nil {
					log.Fatal(err)
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
					log.Fatal(err)
				}
				return nil
			},
		},
		command.NewLsDistributionsCommand(client),
		whichCmd,
		command.NewInitCommand(),
	)
	rootCmd.Flags().Bool("version", false, "version of javm")
	rootCmd.PersistentFlags().String("fd3", "", "")
	rootCmd.PersistentFlags().MarkHidden("fd3")
	if err := rootCmd.Execute(); err != nil {
		os.Exit(-1)
	}
}

func parseTrimTo(value string) semver.VersionPart {
	switch strings.ToLower(value) {
	case "major":
		return semver.VPMajor
	case "minor":
		return semver.VPMinor
	case "patch":
		return semver.VPPatch
	default:
		log.Fatal("Unexpected value of --latest (must be either \"major\", \"minor\" or \"patch\")")
		return -1
	}
}

type jabbarc struct {
	JDK string
}

func rc() (rc jabbarc) {
	b, err := os.ReadFile(".jabbarc")
	if err != nil {
		return
	}
	// content can be a string (jdk version)
	err = yaml.Unmarshal(b, &rc.JDK)
	if err != nil {
		// or a struct
		err = yaml.Unmarshal(b, &rc)
		if err != nil {
			log.Fatal(".jabbarc is not valid")
		}
	}
	return
}

func use(ver string) error {
	out, err := command.Use(ver)
	if err != nil {
		log.Fatal(err)
	}
	printForShellToEval(out)
	return nil
}

func printForShellToEval(out []string) {
	fd3, _ := rootCmd.Flags().GetString("fd3")
	if fd3 != "" {
		os.WriteFile(fd3, []byte(strings.Join(out, "\n")), 0666)
	} else {
		fd3 := os.NewFile(3, "fd3")
		for _, line := range out {
			fmt.Fprintln(fd3, line)
		}
	}
}
