package command

import (
	"bytes"
	"encoding/base64"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

const fakeEnvironmentFile = "JAVM_TEST_FAKE_ENVIRONMENT_FILE"

// The test binary doubles as the javm executable that the wrapper runs: it
// writes the chosen --fd3 content and exits.
func init() {
	encoded, ok := os.LookupEnv(fakeEnvironmentFile)
	if !ok {
		return
	}
	for i, arg := range os.Args {
		if arg == "--fd3" && i+1 < len(os.Args) {
			content, err := base64.StdEncoding.DecodeString(encoded)
			if err != nil {
				os.Exit(70)
			}
			if err := os.WriteFile(os.Args[i+1], content, 0o600); err != nil {
				os.Exit(71)
			}
		}
	}
	os.Exit(0)
}

type cmdWrapperResult struct {
	exit, path, javaHome, stderr string
}

func runCmdWrapper(t *testing.T, records string, defaultInitialized bool, args string) cmdWrapperResult {
	t.Helper()
	t.Setenv("JAVM_HOME", t.TempDir())
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	previous := getExecutablePath
	getExecutablePath = func() (string, error) { return filepath.ToSlash(executable), nil }
	t.Cleanup(func() { getExecutablePath = previous })

	init := NewInitCommand()
	var script bytes.Buffer
	init.SetOut(&script)
	init.SetArgs([]string{"cmd"})
	if err := init.Execute(); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	wrapper := filepath.Join(dir, "javm.cmd")
	if err := os.WriteFile(wrapper, script.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}

	system32 := filepath.Join(os.Getenv("SystemRoot"), "System32")
	command := exec.Command(filepath.Join(system32, "cmd.exe"))
	command.SysProcAttr = &syscall.SysProcAttr{CmdLine: `cmd.exe /d /v:on /c "call "` + wrapper + `" ` + args +
		` & echo EXIT=[!ERRORLEVEL!] & echo PATH=[!PATH!] & echo JAVA_HOME=[!JAVA_HOME!]"`}
	env := []string{
		"SystemRoot=" + os.Getenv("SystemRoot"),
		"TEMP=" + dir,
		"PATH=" + system32 + `;C:\original`,
		`JAVA_HOME=C:\original-java`,
		fakeEnvironmentFile + "=" + base64.StdEncoding.EncodeToString([]byte(records)),
	}
	if defaultInitialized {
		env = append(env, "_JAVM_DEFAULT_INITIALIZED=1")
	}
	command.Env = env
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	if err := command.Run(); err != nil {
		t.Fatalf("cmd.exe: %v\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String())
	}
	result := cmdWrapperResult{stderr: stderr.String()}
	for line := range strings.Lines(stdout.String()) {
		line = strings.TrimSpace(line)
		for prefix, field := range map[string]*string{"EXIT=[": &result.exit, "PATH=[": &result.path, "JAVA_HOME=[": &result.javaHome} {
			if value, ok := strings.CutPrefix(line, prefix); ok {
				value = strings.TrimSuffix(value, "]")
				// Delayed expansion leaves an undefined variable as !NAME!.
				if value == "!PATH!" || value == "!JAVA_HOME!" {
					value = ""
				}
				*field = value
			}
		}
	}
	return result
}

func TestCmdWrapperAppliesEnvironmentRecords(t *testing.T) {
	originalPath := filepath.Join(os.Getenv("SystemRoot"), "System32") + `;C:\original`
	tests := []struct {
		name               string
		records            string
		defaultInitialized bool
		args               string
		want               cmdWrapperResult
	}{
		{
			name: "complete file",
			records: "SET\tPATH\tC:\\jdk21\\bin;" + originalPath + "\nSET\tJAVA_HOME\tC:\\jdk21\n" +
				"SET\tJAVM_ACTIVE_JAVA_HOME\tC:\\jdk21\nSET\tJAVA_HOME_BEFORE_JAVM\tC:\\original-java\n",
			defaultInitialized: true, args: "use 21",
			want: cmdWrapperResult{exit: "0", path: `C:\jdk21\bin;` + originalPath, javaHome: `C:\jdk21`},
		},
		{
			name: "empty file", defaultInitialized: true, args: "use 21",
			want: cmdWrapperResult{exit: "1", path: originalPath, javaHome: `C:\original-java`,
				stderr: "javm: environment update file is incomplete."},
		},
		{
			name: "no JAVA_HOME record", records: "SET\tPATH\tC:\\other\n",
			defaultInitialized: true, args: "deactivate",
			want: cmdWrapperResult{exit: "0", path: `C:\other`, javaHome: `C:\original-java`},
		},
		{
			name: "empty JAVA_HOME value", records: "SET\tPATH\tC:\\other\nSET\tJAVA_HOME\t\n",
			defaultInitialized: true, args: "deactivate",
			want: cmdWrapperResult{exit: "0", path: `C:\other`, javaHome: ""},
		},
		{
			name: "startup default with empty file", args: "ls",
			want: cmdWrapperResult{exit: "0", path: originalPath, javaHome: `C:\original-java`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := runCmdWrapper(t, tt.records, tt.defaultInitialized, tt.args)
			if got.exit != tt.want.exit || got.path != tt.want.path || got.javaHome != tt.want.javaHome ||
				strings.TrimSpace(got.stderr) != tt.want.stderr {
				t.Fatalf("got %+v\nwant %+v", got, tt.want)
			}
		})
	}
}
