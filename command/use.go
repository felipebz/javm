package command

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
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
				// A broken default must not make every new shell fail; the
				// shell wrappers apply nothing when no records are written.
				if errors.Is(err, ErrUsage) {
					loggerFromContext(cmd.Context()).Warnf("ignoring %s: %v; run 'javm default <version>' to replace it",
						sanitizeTerminalText(defaultVersionPath()), err)
					return nil
				}
				if err != nil {
					return err
				}
			} else if len(args) == 0 {
				selector, err := javaVersionSelector()
				if err != nil {
					return UsageError(err)
				}
				ver = selector
			} else {
				ver = args[0]
			}
			fd3, _ := cmd.Flags().GetString("fd3")

			out, err := UseContext(cmd.Context(), ver)
			if useDefault && errors.Is(err, ErrNotFound) {
				loggerFromContext(cmd.Context()).Warnf("default JDK %q is not installed; run 'javm install %s' or 'javm default <version>'",
					sanitizeTerminalText(ver), sanitizeTerminalText(ver))
				return nil
			}
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
	return useRecords(path, os.Environ())
}

func useRecords(path string, env []string) ([]string, error) {
	selected, err := buildJDKEnvironment(path, env)
	if err != nil {
		return nil, err
	}
	records := []string{
		"SET\tPATH\t" + selected.Path,
		"SET\tJAVA_HOME\t" + selected.JavaHome,
		"SET\t" + activeJavaHomeKey + "\t" + selected.JavaHome,
	}
	if original, ok := originalJavaHome(env); ok {
		return append(records, "SET\t"+javaHomeBeforeJavmKey+"\t"+original), nil
	}
	return append(records, "UNSET\t"+javaHomeBeforeJavmKey), nil
}

// originalJavaHome returns the JAVA_HOME the user had before javm activated
// the shell; ok is false when there was none.
func originalJavaHome(env []string) (string, bool) {
	if _, active := activeJavaHome(env); active {
		return lookupEnvironmentValue(env, javaHomeBeforeJavmKey)
	}
	return lookupEnvironmentValue(env, "JAVA_HOME")
}
