package command

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felipebz/javm/discoapi"
	log "github.com/sirupsen/logrus"
)

type failingShellWriter struct{}

func (failingShellWriter) Write([]byte) (int, error) {
	return 0, errors.New("The handle is invalid")
}

func TestWriteShellEnvironmentReportsMissingIntegration(t *testing.T) {
	err := writeShellEnvironment(failingShellWriter{}, []string{"SET\tJAVA_HOME\tC:\\Java"})
	if !errors.Is(err, ErrShellIntegration) {
		t.Fatalf("writeShellEnvironment() error = %v, want ErrShellIntegration", err)
	}
	if !strings.HasPrefix(err.Error(), ErrShellIntegration.Error()+";") {
		t.Fatalf("writeShellEnvironment() error = %q, want integration guidance", err)
	}
}

func TestShellEnvironmentRejectsInvalidRecordsBeforeWriting(t *testing.T) {
	tests := []struct {
		name   string
		record string
	}{
		{name: "empty record", record: ""},
		{name: "unknown operation", record: "EXPORT\tJAVA_HOME\t/opt/java"},
		{name: "lowercase operation", record: "set\tJAVA_HOME\t/opt/java"},
		{name: "missing fields", record: "SET"},
		{name: "missing SET value", record: "SET\tJAVA_HOME"},
		{name: "extra SET field", record: "SET\tJAVA_HOME\t/opt/java\tinjected"},
		{name: "empty extra SET field", record: "SET\tJAVA_HOME\t/opt/java\t"},
		{name: "empty SET key", record: "SET\t\t/opt/java"},
		{name: "missing UNSET key", record: "UNSET"},
		{name: "empty UNSET key", record: "UNSET\t"},
		{name: "extra UNSET field", record: "UNSET\tJAVA_HOME\tinjected"},
		{name: "empty extra UNSET field", record: "UNSET\tJAVA_HOME\t"},
		{name: "tab in SET key", record: "SET\tJAVA\tHOME\t/opt/java"},
		{name: "tab in SET value", record: "SET\tJAVA_HOME\t/opt\t/java"},
		{name: "tab in UNSET key", record: "UNSET\tJAVA\tHOME"},
		{name: "newline record injection", record: "SET\tJAVA_HOME\t/opt/java\nUNSET\tPATH"},
		{name: "carriage return record injection", record: "SET\tJAVA_HOME\t/opt/java\rUNSET\tPATH"},
		{name: "CRLF record injection", record: "SET\tJAVA_HOME\t/opt/java\r\nUNSET\tPATH"},
	}
	for _, control := range []struct {
		name  string
		value string
	}{
		{name: "CR", value: "\r"},
		{name: "LF", value: "\n"},
		{name: "NUL", value: "\x00"},
	} {
		tests = append(tests,
			struct {
				name   string
				record string
			}{name: control.name + " in SET key", record: "SET\tJAVA" + control.value + "_HOME\t/opt/java"},
			struct {
				name   string
				record string
			}{name: control.name + " in SET value", record: "SET\tJAVA_HOME\t/opt/" + control.value + "java"},
			struct {
				name   string
				record string
			}{name: control.name + " in UNSET key", record: "UNSET\tJAVA" + control.value + "_HOME"},
		)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			records := []string{"SET\tJAVA_HOME\t/opt/valid", tt.record, "UNSET\tJAVM_CURRENT"}
			assertInvalid := func(err error) {
				t.Helper()
				if err == nil || !strings.Contains(err.Error(), "invalid shell environment record 2:") {
					t.Fatalf("error = %v, want validation error identifying record 2", err)
				}
				if errors.Is(err, ErrShellIntegration) {
					t.Fatalf("validation error = %v, must not report missing shell integration", err)
				}
			}

			t.Run("writer", func(t *testing.T) {
				var output bytes.Buffer
				assertInvalid(writeShellEnvironment(&output, records))
				if output.Len() != 0 {
					t.Fatalf("invalid records wrote partial output: %q", output.String())
				}
			})

			t.Run("existing file", func(t *testing.T) {
				path := filepath.Join(t.TempDir(), "environment")
				const original = "existing shell environment\n"
				if err := os.WriteFile(path, []byte(original), 0600); err != nil {
					t.Fatal(err)
				}
				assertInvalid(printForShellToEval(records, path))
				got, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if string(got) != original {
					t.Fatalf("invalid records changed existing file to %q", got)
				}
			})

			t.Run("new file", func(t *testing.T) {
				path := filepath.Join(t.TempDir(), "environment")
				assertInvalid(printForShellToEval(records, path))
				if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("invalid records created output file; stat error = %v", err)
				}
			})
		})
	}
}

func TestShellEnvironmentPreservesValidRecordFormat(t *testing.T) {
	tests := []struct {
		name       string
		records    []string
		wantWriter string
		wantFile   string
	}{
		{name: "nil records"},
		{name: "empty records", records: []string{}},
		{
			name:       "empty SET value",
			records:    []string{"SET\tJAVA_HOME\t"},
			wantWriter: "SET\tJAVA_HOME\t\n",
			wantFile:   "SET\tJAVA_HOME\t",
		},
		{
			name:       "SET and UNSET records",
			records:    []string{"SET\tJAVA_HOME\tC:\\Program Files\\Java", "SET\tJAVM_CURRENT\t", "UNSET\tJAVM_PREVIOUS"},
			wantWriter: "SET\tJAVA_HOME\tC:\\Program Files\\Java\nSET\tJAVM_CURRENT\t\nUNSET\tJAVM_PREVIOUS\n",
			wantFile:   "SET\tJAVA_HOME\tC:\\Program Files\\Java\nSET\tJAVM_CURRENT\t\nUNSET\tJAVM_PREVIOUS",
		},
		{
			name:       "literal shell punctuation and Unicode",
			records:    []string{"SET\tJAVA_HOME\t/opt/Java café; '$HOME' \"quoted\""},
			wantWriter: "SET\tJAVA_HOME\t/opt/Java café; '$HOME' \"quoted\"\n",
			wantFile:   "SET\tJAVA_HOME\t/opt/Java café; '$HOME' \"quoted\"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			if err := writeShellEnvironment(&output, tt.records); err != nil {
				t.Fatal(err)
			}
			if got := output.String(); got != tt.wantWriter {
				t.Fatalf("writer output = %q, want %q", got, tt.wantWriter)
			}

			path := filepath.Join(t.TempDir(), "environment")
			if err := printForShellToEval(tt.records, path); err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.wantFile {
				t.Fatalf("file output = %q, want %q", got, tt.wantFile)
			}
		})
	}
}

func TestShellIntegrationErrorMessage(t *testing.T) {
	tests := []struct {
		name string
		hint shellIntegrationHint
		ok   bool
		want string
	}{
		{
			name: "detected shell",
			hint: shellIntegrationHint{
				name:    "PowerShell",
				command: `iex "$(javm init pwsh)"`,
			},
			ok:   true,
			want: "shell integration is not active; enable javm for PowerShell with:\niex \"$(javm init pwsh)\"",
		},
		{
			name: "unknown shell",
			want: "shell integration is not active; run `javm init <shell>` and invoke javm through the generated shell wrapper",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shellIntegrationError(tt.hint, tt.ok).Error(); got != tt.want {
				t.Fatalf("shellIntegrationError() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMakePackageIndex(t *testing.T) {
	mock := &mockPackagesClient{
		Pkgs: []discoapi.Package{
			{JavaVersion: "21.0.1+9", Distribution: "temurin", DistributionVersion: "21.0.1"},
			{JavaVersion: "17+35", Distribution: "zulu", DistributionVersion: "17"},
		},
	}
	idx, err := makePackageIndex(context.Background(), mock, "linux", "amd64", "")
	if err != nil {
		t.Fatal(err)
	}

	if !hasPackageWithVersion(idx, "temurin", "21.0.1") {
		t.Errorf("expected to find package temurin@21.0.1")
	}

	if !hasPackageWithVersion(idx, "zulu", "17") {
		t.Errorf("expected to find package zulu@17")
	}

	if len(idx.Sorted) != 2 {
		t.Errorf("expected 2 versions in Sorted")
	}
}

func TestMakePackageIndexDiagnosesInvalidJavaVersionAtDebugLevel(t *testing.T) {
	mock := &mockPackagesClient{Pkgs: []discoapi.Package{
		{Id: "valid", JavaVersion: "25.0.4.1+1", Distribution: "temurin"},
		{Id: "invalid", JavaVersion: "25.0.4.1.invalid", Distribution: "temurin"},
	}}
	var diagnostics bytes.Buffer
	logger := log.New()
	logger.SetOutput(&diagnostics)
	logger.SetLevel(log.DebugLevel)

	index, err := makePackageIndex(WithRuntime(context.Background(), Runtime{Logger: logger, Err: &diagnostics}), mock, "linux", "amd64", "temurin")
	if err != nil {
		t.Fatal(err)
	}
	if len(index.Sorted) != 1 {
		t.Fatalf("valid package count = %d, want 1", len(index.Sorted))
	}
	if !strings.Contains(diagnostics.String(), "invalid Java version") || !strings.Contains(diagnostics.String(), "25.0.4.1.invalid") {
		t.Fatalf("invalid package was not diagnosed: %q", diagnostics.String())
	}
}

func hasPackageWithVersion(idx *packageIndex, distribution, version string) bool {
	for _, pkg := range idx.ByVersion {
		if pkg.Distribution == distribution && strings.HasPrefix(pkg.JavaVersion, version) {
			return true
		}
	}
	return false
}
