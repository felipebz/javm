package command

import "testing"

func TestDetectShellFromEnvironment(t *testing.T) {
	tests := []struct {
		name    string
		goos    string
		env     map[string]string
		want    string
		command string
		found   bool
	}{
		{
			name:    "bash from SHELL",
			goos:    "linux",
			env:     map[string]string{"SHELL": "/bin/bash"},
			want:    "Bash",
			command: `eval "$(javm init bash)"`,
			found:   true,
		},
		{
			name:    "zsh from SHELL",
			goos:    "darwin",
			env:     map[string]string{"SHELL": "zsh"},
			want:    "Zsh",
			command: `eval "$(javm init zsh)"`,
			found:   true,
		},
		{
			name:  "PSModulePath alone does not imply PowerShell",
			goos:  "windows",
			env:   map[string]string{"PSModulePath": `C:\Windows\System32\WindowsPowerShell\v1.0\Modules`},
			found: false,
		},
		{
			name:  "ComSpec alone does not imply Command Prompt",
			goos:  "windows",
			env:   map[string]string{"ComSpec": `C:\Windows\System32\cmd.exe`},
			found: false,
		},
		{
			name:  "SHELL does not infer a Windows shell",
			goos:  "windows",
			env:   map[string]string{"SHELL": "/bin/bash"},
			found: false,
		},
		{
			name:  "unknown shell",
			goos:  "linux",
			env:   map[string]string{"SHELL": "/bin/ksh"},
			found: false,
		},
		{
			name:  "missing environment",
			goos:  "windows",
			env:   map[string]string{},
			found: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hint, found := detectShellFromEnvironment(tt.goos, mapLookup(tt.env))
			if found != tt.found {
				t.Fatalf("detected shell = %v, want found=%v", hint, tt.found)
			}
			if !tt.found {
				return
			}
			if hint.name != tt.want {
				t.Fatalf("shell name = %q, want %q", hint.name, tt.want)
			}
			if hint.command != tt.command {
				t.Fatalf("shell command = %q, want %q", hint.command, tt.command)
			}
		})
	}
}

func TestShellName(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "bash path", value: "/bin/bash", want: "bash"},
		{name: "zsh path", value: "/usr/bin/zsh", want: "zsh"},
		{name: "Windows executable path", value: `C:\Tools\pwsh.exe`, want: "pwsh"},
		{name: "uppercase name", value: "BASH", want: "bash"},
		{name: "surrounding whitespace", value: " /bin/fish ", want: "fish"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shellName(tt.value); got != tt.want {
				t.Fatalf("shellName(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func mapLookup(values map[string]string) func(string) (string, bool) {
	return func(name string) (string, bool) {
		value, ok := values[name]
		return value, ok
	}
}
