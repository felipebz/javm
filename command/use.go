package command

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"

	"github.com/felipebz/javm/cfg"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func NewUseCommand() *cobra.Command {
	cmd := &cobra.Command{
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

			out, err := Use(ver)
			if err != nil {
				return err
			}
			printForShellToEval(out, fd3)
			return nil
		},
		Example: "  javm use 1.8\n" +
			"  javm use ~1.8.73 # same as \">=1.8.73 <1.9.0\"",
	}
	cmd.Flags().String("fd3", "", "")
	_ = cmd.Flags().MarkHidden("fd3")
	return cmd
}

func Use(selector string) ([]string, error) {
	aliasValue := getAlias(selector)
	if aliasValue != "" {
		selector = aliasValue
	}

	jdks, err := Ls(false)
	if err != nil {
		return nil, err
	}
	jdk, err := FindBestMatchJDK(jdks, selector)
	if err != nil {
		return nil, err
	}
	return usePath(jdk.Path)
}

func usePath(path string) ([]string, error) {
	sep := string(os.PathListSeparator)
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	pth, _ := os.LookupEnv("PATH")
	rgxp := regexp.MustCompile(regexp.QuoteMeta(filepath.Join(cfg.Dir(), "jdk")) + "[^" + sep + "]+[" + sep + "]")
	// strip references to ~/.jabba/jdk/*, otherwise leave unchanged
	pth = rgxp.ReplaceAllString(pth, "")
	if runtime.GOOS == "darwin" {
		path = filepath.Join(path, "Contents", "Home")
	}
	systemJavaHome, overrideWasSet := os.LookupEnv("JAVA_HOME_BEFORE_JABBA")
	if !overrideWasSet {
		systemJavaHome, _ = os.LookupEnv("JAVA_HOME")
	}
	return []string{
		"SET\tPATH\t" + filepath.Join(path, "bin") + string(os.PathListSeparator) + pth,
		"SET\tJAVA_HOME\t" + path,
		"SET\tJAVA_HOME_BEFORE_JABBA\t" + systemJavaHome,
	}, nil
}
