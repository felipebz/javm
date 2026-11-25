package command

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felipebz/javm/cfg"
	"github.com/felipebz/javm/discoapi"
	"github.com/felipebz/javm/discovery"
)

func TestRunInstallFollowsEmergencyReleaseOfPlainSelector(t *testing.T) {
	archive := makeZipArchive(t, []zipTestEntry{{name: javaArchivePath(), body: "java", mode: 0755}})
	for _, tc := range []struct {
		selector, wantVersion, wantPackage string
	}{
		{"temurin@25.0.5", "temurin@25.0.5.1+2", "emergency"},
		{"temurin@25.0.5.0", "temurin@25.0.5+8", "base"},
	} {
		client := &versionSelectionClient{
			packages: []discoapi.Package{
				{Id: "base", JavaVersion: "25.0.5+8", Distribution: "temurin", DistributionVersion: "25.0.5"},
				{Id: "emergency", JavaVersion: "25.0.5.1+2", Distribution: "temurin", DistributionVersion: "25.0.5.1"},
			},
			archive: archive,
		}
		version, err := runInstall(context.Background(), client, tc.selector, filepath.Join(t.TempDir(), "jdk"))
		if err != nil {
			t.Fatalf("runInstall(%q) error = %v", tc.selector, err)
		}
		if version != tc.wantVersion || client.infoID != tc.wantPackage {
			t.Fatalf("runInstall(%q) chose %q (package %q), want %q (package %q)", tc.selector, version, client.infoID, tc.wantVersion, tc.wantPackage)
		}
	}
}

func TestFindBestMatchJDKFollowsEmergencyReleaseOfPlainSelector(t *testing.T) {
	jdks := []discovery.JDK{
		{Identifier: "temurin@25.0.5", Version: "25.0.5+8", Source: "javm"},
		{Identifier: "temurin@25.0.5.1", Version: "25.0.5.1+2", Source: "javm"},
	}
	for selector, want := range map[string]string{"25.0.5": "temurin@25.0.5.1", "25.0.5.0": "temurin@25.0.5"} {
		got, err := FindBestMatchJDK(jdks, selector)
		if err != nil {
			t.Fatalf("FindBestMatchJDK(%q) error = %v", selector, err)
		}
		if got.Identifier != want {
			t.Fatalf("FindBestMatchJDK(%q) = %q, want %q", selector, got.Identifier, want)
		}
	}
}

func TestUninstallPlainSelectorMatchingEmergencyReleaseIsAmbiguous(t *testing.T) {
	setupUninstallPins(t, map[string]string{"temurin@25.0.5": "25.0.5+8", "temurin@25.0.5.1": "25.0.5.1+2"}, nil)

	cmd := NewUninstallCommand()
	cmd.SetArgs([]string{"25.0.5"})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	err := cmd.ExecuteContext(context.Background())
	if !errors.Is(err, ErrUsage) || !strings.Contains(err.Error(), "matches 2 installed JDKs") || !strings.Contains(err.Error(), "pass the exact identifier") {
		t.Fatalf("uninstall 25.0.5 error = %v, want the ambiguity usage error", err)
	}
	for _, id := range []string{"temurin@25.0.5", "temurin@25.0.5.1"} {
		if _, err := os.Stat(filepath.Join(cfg.Dir(), "jdk", id)); err != nil {
			t.Fatalf("%s was removed by an ambiguous uninstall: %v", id, err)
		}
	}

	uninstallLogged(t, "temurin@25.0.5")
	if _, err := os.Stat(filepath.Join(cfg.Dir(), "jdk", "temurin@25.0.5")); !os.IsNotExist(err) {
		t.Fatalf("temurin@25.0.5 still exists: %v", err)
	}
	if _, err := os.Stat(filepath.Join(cfg.Dir(), "jdk", "temurin@25.0.5.1")); err != nil {
		t.Fatalf("temurin@25.0.5.1 was removed with temurin@25.0.5: %v", err)
	}
}
