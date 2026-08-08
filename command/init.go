package command

import (
	_ "embed"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"

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

			executable, err = escapeExecutablePath(shell, executable)
			if err != nil {
				return err
			}
			script = strings.NewReplacer("::JAVM::", executable).Replace(script)

			_, defaultErr := readDefaultVersion()
			defaultConfigured := defaultErr == nil
			if defaultErr != nil && !os.IsNotExist(defaultErr) {
				return defaultErr
			}
			if shell == "cmd" {
				defaultInit := strings.Join([]string{
					`if /i "%~1"=="use" goto javm_dispatch`,
					`if /i "%~1"=="deactivate" goto javm_dispatch`,
					`if not defined _JAVM_DEFAULT_INITIALIZED goto javm_initialize_default`,
				}, "\n")
				script = strings.ReplaceAll(script, "::JAVM_DEFAULT_INIT::", defaultInit)
			} else if defaultConfigured {
				script += "\njavm use --default\n"
			}

			if shell == "pwsh" || shell == "powershell" {
				scriptPath, err := writePowerShellInitScript(script)
				if err != nil {
					return err
				}
				scriptPath, err = escapeExecutablePath(shell, scriptPath)
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

func escapeExecutablePath(shell, value string) (string, error) {
	if strings.ContainsAny(value, "\r\n\x00") {
		return "", fmt.Errorf("executable path contains a forbidden control character")
	}
	switch shell {
	case "cmd":
		return escapeBatchValue(value), nil
	case "powershell", "pwsh":
		return strings.ReplaceAll(value, "'", "''"), nil
	case "fish":
		return strings.NewReplacer(
			`\`, `\\`,
			`"`, `\"`,
			`$`, `\$`,
			`(`, `\(`,
			`)`, `\)`,
		).Replace(value), nil
	case "nu":
		return strings.NewReplacer(
			`\`, `\\`,
			`"`, `\"`,
		).Replace(value), nil
	default:
		return strings.NewReplacer(
			`\`, `\\`,
			`"`, `\"`,
			`$`, `\$`,
			"`", "\\`",
		).Replace(value), nil
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
