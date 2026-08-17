package command

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/felipebz/javm/discoapi"
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

func hasPackageWithVersion(idx *packageIndex, distribution, version string) bool {
	for _, pkg := range idx.ByVersion {
		if pkg.Distribution == distribution && strings.HasPrefix(pkg.JavaVersion, version) {
			return true
		}
	}
	return false
}
