package command

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/felipebz/javm/cfg"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func NewUseCommand() *cobra.Command {
	var useDefault bool
	cmd := &cobra.Command{
		Use:   "use [version to use]",
		Short: "Modify PATH & JAVA_HOME to use specific JDK",
		Args:  UsageArgs(cobra.MaximumNArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			var ver string
			if useDefault {
				if len(args) != 0 {
					return UsageError(fmt.Errorf("--default cannot be combined with a version argument"))
				}
				var err error
				ver, err = readDefaultVersion()
				if errors.Is(err, os.ErrNotExist) {
					return nil
				}
				if err != nil {
					return err
				}
			} else if len(args) == 0 {
				ver = cfg.ReadJavaVersion()
				if ver == "" {
					return pflag.ErrHelp
				}
			} else {
				ver = args[0]
			}
			fd3, _ := cmd.Flags().GetString("fd3")

			out, err := UseContext(cmd.Context(), ver)
			if err != nil {
				return err
			}
			return printForShellToEval(out, fd3)
		},
		Example: "  javm use 1.8\n" +
			"  javm use ~1.8.73 # same as \">=1.8.73 <1.9.0\"",
	}
	cmd.Flags().String("fd3", "", "")
	_ = cmd.Flags().MarkHidden("fd3")
	cmd.Flags().BoolVar(&useDefault, "default", false, "use the configured default version")
	_ = cmd.Flags().MarkHidden("default")
	return cmd
}

func Use(selector string) ([]string, error) {
	return UseContext(context.Background(), selector)
}

func UseContext(ctx context.Context, selector string) ([]string, error) {
	jdk, err := resolveJDKContext(ctx, selector)
	if err != nil {
		return nil, err
	}
	return usePath(jdk.Path)
}

func usePath(path string) ([]string, error) {
	env := os.Environ()
	selected, err := buildJDKEnvironment(path, env)
	if err != nil {
		return nil, err
	}
	systemJavaHome, overrideWasSet := lookupEnvironmentValue(env, "JAVA_HOME_BEFORE_JAVM")
	if !overrideWasSet {
		systemJavaHome, _ = lookupEnvironmentValue(env, "JAVA_HOME")
	}
	return []string{
		"SET\tPATH\t" + selected.Path,
		"SET\tJAVA_HOME\t" + selected.JavaHome,
		"SET\tJAVA_HOME_BEFORE_JAVM\t" + systemJavaHome,
	}, nil
}
