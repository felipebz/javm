package command

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
)

func NewWhichCommand() *cobra.Command {
	var whichHome bool
	cmd := &cobra.Command{
		Use:   "which [version]",
		Short: "Display path to installed JDK",
		Args:  UsageArgs(cobra.MaximumNArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			var ver string
			if len(args) == 0 {
				selector, err := javaVersionSelector()
				if err != nil {
					return UsageError(err)
				}
				ver = selector
			} else {
				ver = args[0]
			}
			dir, err := WhichContext(cmd.Context(), ver, whichHome)
			if err != nil {
				return err
			}
			if dir != "" {
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), sanitizeTerminalText(dir)); err != nil {
					return fmt.Errorf("write JDK path: %w", err)
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&whichHome, "home", false, "Account for platform differences so that value could be used as JAVA_HOME (e.g. append \"/Contents/Home\" on macOS)")
	return cmd
}

func Which(selector string, home bool) (string, error) {
	return WhichContext(context.Background(), selector, home)
}

func WhichContext(ctx context.Context, selector string, home bool) (string, error) {
	jdk, err := resolveJDKContext(ctx, selector)
	if err != nil {
		return "", err
	}
	path := jdk.Path
	if home && runtime.GOOS == "darwin" {
		path = filepath.Join(path, "Contents", "Home")
	}
	return path, nil
}
