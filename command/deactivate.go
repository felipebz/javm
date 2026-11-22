package command

import (
	"os"

	"github.com/spf13/cobra"
)

func NewDeactivateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deactivate",
		Short: "Undo javm's changes to the current shell",
		Args:  UsageArgs(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := deactivateRecords(os.Environ())
			fd3, _ := cmd.Flags().GetString("fd3")
			return printForShellToEval(out, fd3)
		},
	}
	cmd.Flags().String("fd3", "", "")
	_ = cmd.Flags().MarkHidden("fd3")
	return cmd
}

// deactivateRecords undoes exactly what use changed: the PATH entries javm
// added and, when javm is active, JAVA_HOME. An inactive shell keeps its
// JAVA_HOME.
func deactivateRecords(env []string) []string {
	active, isActive := activeJavaHome(env)
	pathValue, _ := lookupEnvironmentValue(env, "PATH")
	remaining, _ := stripManagedJDKPathsWithPresence(pathValue, active)
	records := []string{"SET\tPATH\t" + remaining}
	if isActive {
		if original, ok := lookupEnvironmentValue(env, javaHomeBeforeJavmKey); ok {
			records = append(records, "SET\tJAVA_HOME\t"+original)
		} else {
			records = append(records, "UNSET\tJAVA_HOME")
		}
	}
	return append(records,
		"UNSET\t"+javaHomeBeforeJavmKey,
		"UNSET\t"+activeJavaHomeKey,
	)
}
