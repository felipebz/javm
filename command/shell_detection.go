package command

import (
	"os"
	"runtime"
	"strings"
)

type shellIntegrationHint struct {
	name    string
	command string
}

func detectedShellHint() (shellIntegrationHint, bool) {
	if hint, ok := parentShellHint(); ok {
		return hint, true
	}
	return detectShellFromEnvironment(runtime.GOOS, os.LookupEnv)
}

func detectShellFromEnvironment(goos string, lookup func(string) (string, bool)) (shellIntegrationHint, bool) {
	if goos != "windows" {
		if value, ok := lookup("SHELL"); ok {
			return shellIntegrationHintForName(shellName(value))
		}
	}
	return shellIntegrationHint{}, false
}

func shellName(value string) string {
	value = strings.TrimSpace(value)
	if index := strings.LastIndexAny(value, `/\`); index >= 0 {
		value = value[index+1:]
	}
	value = strings.ToLower(value)
	return strings.TrimSuffix(value, ".exe")
}

func shellIntegrationHintForName(name string) (shellIntegrationHint, bool) {
	switch name {
	case "bash":
		return shellIntegrationHint{name: "Bash", command: `eval "$(javm init bash)"`}, true
	case "zsh":
		return shellIntegrationHint{name: "Zsh", command: `eval "$(javm init zsh)"`}, true
	case "fish":
		return shellIntegrationHint{name: "Fish", command: `javm init fish | source`}, true
	case "nu", "nushell":
		return shellIntegrationHint{
			name:    "Nushell",
			command: "javm init nu | save -f ~/.local/share/javm/javm.nu\nsource ~/.local/share/javm/javm.nu",
		}, true
	case "powershell", "pwsh":
		return shellIntegrationHint{name: "PowerShell", command: `iex "$(javm init pwsh)"`}, true
	case "cmd", "command":
		return shellIntegrationHint{
			name: "Command Prompt",
			command: "mkdir \"%LOCALAPPDATA%\\javm\\cmd\" 2>NUL\n" +
				"javm init cmd > \"%LOCALAPPDATA%\\javm\\cmd\\javm.cmd\"\n" +
				"set \"PATH=%LOCALAPPDATA%\\javm\\cmd;%PATH%\"",
		}, true
	default:
		return shellIntegrationHint{}, false
	}
}
