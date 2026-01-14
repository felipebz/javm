package command

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/felipebz/javm/cfg"
	"github.com/spf13/cobra"
)

var lookPath = exec.LookPath

func NewCurrentCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: "Display currently 'use'ed version",
		Run: func(cmd *cobra.Command, args []string) {
			ver := current()
			if ver != "" {
				fmt.Println(ver)
			}
		},
	}
}

func current() string {
	javaPath, err := lookPath("java")
	if err == nil {
		prefix := filepath.Join(cfg.Dir(), "jdk") + string(os.PathSeparator)
		if strings.HasPrefix(javaPath, prefix) {
			index := strings.Index(javaPath[len(prefix):], string(os.PathSeparator))
			if index != -1 {
				return javaPath[len(prefix) : len(prefix)+index]
			}
		}
	}
	return ""
}
