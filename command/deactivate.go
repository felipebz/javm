package command

import (
	"os"
	"path/filepath"
	"regexp"

	"github.com/felipebz/javm/cfg"
	"github.com/spf13/cobra"
)

func NewDeactivateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deactivate",
		Short: "Undo effects of `javm` on current shell",
		RunE: func(cmd *cobra.Command, args []string) error {
			out, err := deactivate()
			if err != nil {
				return err
			}
			fd3, _ := cmd.Flags().GetString("fd3")
			printForShellToEval(out, fd3)
			return nil
		},
	}
	cmd.Flags().String("fd3", "", "")
	_ = cmd.Flags().MarkHidden("fd3")
	return cmd
}

func deactivate() ([]string, error) {
	sep := string(os.PathListSeparator)
	pth, _ := os.LookupEnv("PATH")
	rgxp := regexp.MustCompile(regexp.QuoteMeta(filepath.Join(cfg.Dir(), "jdk")) + "[^" + sep + "]+[" + sep + "]")
	// strip references to managed jdks dir, otherwise leave unchanged
	pth = rgxp.ReplaceAllString(pth, "")
	javaHome, overrideWasSet := os.LookupEnv("JAVA_HOME_BEFORE_JAVM")
	if !overrideWasSet {
		javaHome, _ = os.LookupEnv("JAVA_HOME")
	}
	return []string{
		"SET\tPATH\t" + pth,
		"SET\tJAVA_HOME\t" + javaHome,
		"UNSET\tJAVA_HOME_BEFORE_JAVM",
	}, nil
}
