package command

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felipebz/javm/discoapi"
)

func TestLegacySelectorHint(t *testing.T) {
	for _, selector := range []string{"1.8", "1.8.0", "~1.8.73", "zulu@1.8", ">=1.7 <1.9"} {
		if hint := legacySelectorHint(selector); !strings.Contains(hint, `written without "1."`) {
			t.Errorf("legacySelectorHint(%q) = %q, want the legacy version hint", selector, hint)
		}
	}
	for _, selector := range []string{"8", "21", "temurin@21", "11.0.1", ""} {
		if hint := legacySelectorHint(selector); hint != "" {
			t.Errorf("legacySelectorHint(%q) = %q, want no hint", selector, hint)
		}
	}
}

func TestRunInstallHintsLegacySelector(t *testing.T) {
	t.Setenv("JAVM_HOME", t.TempDir())
	client := &versionSelectionClient{packages: []discoapi.Package{
		{Id: "eight", JavaVersion: "8.0.502+9", Distribution: "temurin", DistributionVersion: "8.0.502"},
	}}

	_, err := runInstall(context.Background(), client, "1.8", filepath.Join(t.TempDir(), "jdk"))
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("runInstall(1.8) error = %v, want a not-found error", err)
	}
	if !strings.Contains(err.Error(), `No compatible version found for 1.8 (Java 8 and earlier are written without "1.", for example 8)`) {
		t.Fatalf("runInstall(1.8) error = %q, want the legacy version hint", err)
	}
	if client.infoID != "" {
		t.Fatalf("runInstall(1.8) fetched package %q, want no match", client.infoID)
	}
}
