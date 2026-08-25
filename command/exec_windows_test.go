//go:build windows

package command

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felipebz/javm/discovery"
)

func TestExecWindowsRunsBatchWrappers(t *testing.T) {
	argsToPreserve := []string{
		"value with spaces",
		"a&b",
		"a|b",
		"100%",
		"a^b",
		"a>b",
		"a<b",
		"quote\"inside",
		"a!b",
	}

	for _, extension := range []string{".cmd", ".bat"} {
		t.Run(extension, func(t *testing.T) {
			home := filepath.Join(t.TempDir(), "jdk home")
			root := filepath.Join(home, "jdk", "temurin@21.0.1")
			bin := filepath.Join(root, "bin")
			if err := os.MkdirAll(bin, 0o755); err != nil {
				t.Fatal(err)
			}

			probe := filepath.Join(bin, "probe"+extension)
			helper := quoteCmdArgument(os.Args[0]) + " -test.run=TestExecWindowsArgumentHelper -- %*"
			script := "@echo off\r\n" +
				helper + "\r\n" +
				"exit /b %ERRORLEVEL%\r\n"
			if err := os.WriteFile(probe, []byte(script), 0o644); err != nil {
				t.Fatal(err)
			}

			t.Setenv("JAVM_HOME", home)
			t.Setenv("PATH", bin)
			t.Setenv("PATHEXT", extension)
			t.Setenv("JAVM_EXEC_ARGUMENT_HELPER", "1")
			cleanup := setupMockLs()
			t.Cleanup(cleanup)
			mockLsResult = []discovery.JDK{{
				Identifier: "temurin@21.0.1",
				Version:    "21.0.1",
				Source:     "javm",
				Path:       root,
			}}

			var out bytes.Buffer
			cmd := NewExecCommand()
			cmd.SetOut(&out)
			cmdArgs := append([]string{"21", "--", "probe"}, argsToPreserve...)
			cmd.SetArgs(cmdArgs)
			if err := cmd.Execute(); err != nil {
				t.Fatalf("exec %s failed: %v", extension, err)
			}
			output := strings.ReplaceAll(out.String(), "\r\n", "\n")
			wantArgs := fmt.Sprintf("ARGS=%q\n", argsToPreserve)
			if !strings.Contains(output, wantArgs) {
				t.Fatalf("batch did not preserve arguments: got %s, want %s", output, wantArgs)
			}
		})
	}
}

func TestExecWindowsArgumentHelper(t *testing.T) {
	if os.Getenv("JAVM_EXEC_ARGUMENT_HELPER") != "1" {
		t.Skip("helper mode")
	}

	separator := -1
	for index, arg := range os.Args {
		if arg == "--" {
			separator = index
			break
		}
	}
	if separator < 0 {
		t.Fatal("argument separator not found")
	}
	fmt.Printf("ARGS=%q\n", os.Args[separator+1:])
}
