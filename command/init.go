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

//go:embed shellscripts/javm.cmd
var cmdInitScript string

var shellScripts = map[string]string{
	"powershell": pwshInitScript,
	"pwsh":       pwshInitScript,
	"bash":       bashInitScript,
	"zsh":        bashInitScript,
	"fish":       fishInitScript,
	"nu":         nuInitScript,
	"cmd":        cmdInitScript,
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

			if shell == "cmd" {
				executable = escapeBatchValue(executable)
			}
			script = strings.NewReplacer("::JAVM::", executable).Replace(script)

			defaultFile := filepath.Join(cfg.Dir(), "default-version")
			if data, err := os.ReadFile(defaultFile); err == nil && shell != "cmd" {
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
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "& '%s'\n", scriptPath); err != nil {
					return fmt.Errorf("write PowerShell initialization: %w", err)
				}
				return nil
			}

			if _, err := fmt.Fprint(cmd.OutOrStdout(), script); err != nil {
				return fmt.Errorf("write shell initialization: %w", err)
			}
			return nil
		},
	}
}

// escapeBatchValue protects percent signs, which cmd.exe expands even inside a
// quoted SET assignment. Double quotes cannot occur in Windows file names.
func escapeBatchValue(value string) string {
	return strings.ReplaceAll(value, "%", "%%")
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
