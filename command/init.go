package command

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/felipebz/javm/cfg"
	"github.com/spf13/cobra"
)

//go:embed shellscripts/javm.ps1
var pwshInitScript string

//go:embed shellscripts/javm.sh
var bashInitScript string

//go:embed shellscripts/javm.fish
var fishInitScript string

//go:embed shellscripts/javm.nu
var nuInitScript string

var shellScripts = map[string]string{
	"powershell": pwshInitScript,
	"pwsh":       pwshInitScript,
	"bash":       bashInitScript,
	"zsh":        bashInitScript,
	"fish":       fishInitScript,
	"nu":         nuInitScript,
}

var getExecutablePath = realGetExecutablePath
var writePowerShellInitScript = realWritePowerShellInitScript

func NewInitCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "init [shell]",
		Short: "Print shell integration script for javm",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			shell := strings.ToLower(args[0])
			script, ok := shellScripts[shell]
			if !ok {
				return fmt.Errorf("unsupported shell: %s\nSupported shells: %s",
					shell,
					strings.Join(sortedShells(), ", "),
				)
			}

			executable, err := getExecutablePath()
			if err != nil {
				return err
			}

			script = strings.NewReplacer("::JAVM::", executable).Replace(script)

			defaultFile := filepath.Join(cfg.Dir(), "default-version")
			if data, err := os.ReadFile(defaultFile); err == nil {
				ver := strings.TrimSpace(string(data))
				if ver != "" {
					script += "\n" + "javm use " + ver + "\n"
				}
			}

			if shell == "pwsh" || shell == "powershell" {
				scriptPath, err := writePowerShellInitScript(script)
				if err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "& '%s'\n", scriptPath)
				return nil
			}

			fmt.Fprint(cmd.OutOrStdout(), script)
			return nil
		},
	}
}

func realGetExecutablePath() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", err
	}

	if runtime.GOOS == "windows" {
		executable = strings.ReplaceAll(executable, "\\", "/")
	}

	return executable, nil
}

func sortedShells() []string {
	keys := make([]string, 0, len(shellScripts))
	for k := range shellScripts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func realWritePowerShellInitScript(script string) (string, error) {
	tempDir := os.TempDir()
	scriptFile, err := os.CreateTemp(tempDir, "javm-init-*.ps1")
	if err != nil {
		return "", err
	}
	defer scriptFile.Close()

	if _, err := scriptFile.WriteString(script); err != nil {
		return "", err
	}
	return scriptFile.Name(), nil
}
