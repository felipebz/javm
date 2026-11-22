package command

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
)

func uninstallWithLogs(t *testing.T, selector string) (string, error) {
	t.Helper()
	var logs bytes.Buffer
	logger := log.New()
	logger.SetOutput(&logs)
	ctx := WithRuntime(context.Background(), Runtime{Logger: logger, Err: &bytes.Buffer{}})
	cmd := NewUninstallCommand()
	cmd.SetArgs([]string{selector})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SilenceUsage = true
	err := cmd.ExecuteContext(ctx)
	return logs.String(), err
}

func twoTemurin21(t *testing.T) {
	t.Helper()
	setupUninstallPins(t, map[string]string{"temurin@21.0.4": "21.0.4+7", "temurin@21.0.5": "21.0.5+11"}, nil)
}

func TestUninstallRefusesAmbiguousSelector(t *testing.T) {
	twoTemurin21(t)
	_, err := uninstallWithLogs(t, "21")
	if !errors.Is(err, ErrUsage) || !strings.Contains(err.Error(), "21 matches 2 installed JDKs: temurin@21.0.5, temurin@21.0.4") {
		t.Fatalf("uninstall 21 error = %v, want a usage error listing both builds", err)
	}
	assertExists(t, "temurin@21.0.4", true)
	assertExists(t, "temurin@21.0.5", true)
}

func TestUninstallRemovesExactIdentifier(t *testing.T) {
	twoTemurin21(t)
	logs, err := uninstallWithLogs(t, "temurin@21.0.4")
	if err != nil {
		t.Fatal(err)
	}
	assertExists(t, "temurin@21.0.4", false)
	assertExists(t, "temurin@21.0.5", true)
	if !strings.Contains(logs, "Removed temurin@21.0.4") {
		t.Fatalf("logs = %q, want 'Removed temurin@21.0.4'", logs)
	}
}

func TestUninstallRemovesSingleMatch(t *testing.T) {
	setupUninstallPins(t, map[string]string{"temurin@21.0.4": "21.0.4+7"}, nil)
	logs, err := uninstallWithLogs(t, "21")
	if err != nil {
		t.Fatal(err)
	}
	assertExists(t, "temurin@21.0.4", false)
	if !strings.Contains(logs, "Removed temurin@21.0.4") {
		t.Fatalf("logs = %q, want 'Removed temurin@21.0.4'", logs)
	}
}

func TestUninstallRefusesSameVersionFromTwoDistributions(t *testing.T) {
	setupUninstallPins(t, map[string]string{"temurin@21.0.4": "21.0.4+7", "zulu@21.0.4": "21.0.4+7"}, nil)
	_, err := uninstallWithLogs(t, "21.0.4")
	if !errors.Is(err, ErrUsage) || !strings.Contains(err.Error(), "temurin@21.0.4") || !strings.Contains(err.Error(), "zulu@21.0.4") {
		t.Fatalf("uninstall 21.0.4 error = %v, want a usage error listing both", err)
	}
	assertExists(t, "temurin@21.0.4", true)
	assertExists(t, "zulu@21.0.4", true)
}

func TestUninstallExplainsUnmanagedJDK(t *testing.T) {
	t.Setenv("JAVM_HOME", t.TempDir())
	jdk := discoveredJDK(t)
	stubDiscovery(t, jdk)
	_, err := uninstallWithLogs(t, "21")
	if !errors.Is(err, ErrNotFound) || !strings.Contains(err.Error(), "ubuntu-system@21 is not managed by javm") ||
		!strings.Contains(err.Error(), jdk.Path) {
		t.Fatalf("uninstall 21 error = %v, want not-found naming the unmanaged JDK and its path", err)
	}
}

func TestUninstallAmbiguousMessageIsSanitized(t *testing.T) {
	twoTemurin21(t)
	_, err := uninstallWithLogs(t, "21\v")
	if !errors.Is(err, ErrUsage) || strings.ContainsRune(err.Error(), '\v') || !strings.Contains(err.Error(), "matches 2 installed JDKs") {
		t.Fatalf("error = %q, want the ambiguous message without control characters", err)
	}
}
