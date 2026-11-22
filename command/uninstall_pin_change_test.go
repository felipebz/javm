package command

import (
	"bytes"
	"context"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
)

// uninstallInfoLogged runs uninstall with info logs, as without --quiet.
func uninstallInfoLogged(t *testing.T, selector string) string {
	t.Helper()
	var logs bytes.Buffer
	logger := log.New()
	logger.SetOutput(&logs)
	logger.SetLevel(log.InfoLevel)
	logger.SetFormatter(&log.TextFormatter{DisableQuote: true, DisableTimestamp: true})
	cmd := NewUninstallCommand()
	cmd.SetArgs([]string{selector})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	if err := cmd.ExecuteContext(WithRuntime(context.Background(), Runtime{Logger: logger, Err: &bytes.Buffer{}})); err != nil {
		t.Fatalf("uninstall %s error = %v", selector, err)
	}
	return logs.String()
}

func two21Builds() map[string]string {
	return map[string]string{"temurin@21.0.5": "21.0.5+11", "temurin@21.0.4": "21.0.4+7"}
}

func TestUninstallReportsDefaultSelectingAnotherJDK(t *testing.T) {
	setupUninstallPins(t, two21Builds(), map[string]string{"default-version": "21"})

	logs := uninstallInfoLogged(t, "temurin@21.0.5")
	removed := strings.Index(logs, "Removed temurin@21.0.5")
	info := strings.Index(logs, "default 21 now selects temurin@21.0.4")
	if removed < 0 || info < removed || strings.Count(logs, "now selects") != 1 {
		t.Fatalf("logs = %q, want Removed and then one default info line", logs)
	}
}

func TestUninstallReportsAliasSelectingAnotherJDKAfterDefault(t *testing.T) {
	setupUninstallPins(t, two21Builds(), map[string]string{"default-version": "21", "work.alias": "21", "build.alias": "temurin@21"})

	logs := uninstallInfoLogged(t, "temurin@21.0.5")
	want := []string{
		"default 21 now selects temurin@21.0.4",
		"alias build (temurin@21) now selects temurin@21.0.4",
		"alias work (21) now selects temurin@21.0.4",
	}
	last := -1
	for _, line := range want {
		at := strings.Index(logs, line)
		if at <= last {
			t.Fatalf("logs = %q, want %q in order default, then aliases by name", logs, want)
		}
		last = at
	}
}

func TestUninstallIsSilentForPinOfAnotherJDK(t *testing.T) {
	setupUninstallPins(t, map[string]string{"temurin@21.0.5": "21.0.5+11", "temurin@17.0.9": "17.0.9+9"},
		map[string]string{"default-version": "17"})

	if logs := uninstallInfoLogged(t, "temurin@21.0.5"); strings.Contains(logs, "now selects") {
		t.Fatalf("logs = %q, want no info line for a default that kept its JDK", logs)
	}
}

func TestUninstallOnlyWarnsForPinThatLosesItsLastMatch(t *testing.T) {
	setupUninstallPins(t, map[string]string{"temurin@21.0.5": "21.0.5+11"}, map[string]string{"default-version": "21"})

	logs := uninstallInfoLogged(t, "temurin@21.0.5")
	if strings.Contains(logs, "now selects") || strings.Count(logs, "no longer matches an installed JDK") != 1 {
		t.Fatalf("logs = %q, want only the existing warning", logs)
	}
}

func TestUninstallQuietHidesPinChanges(t *testing.T) {
	setupUninstallPins(t, map[string]string{"temurin@21.0.5": "21.0.5+11", "temurin@21.0.4": "21.0.4+7", "temurin@17.0.9": "17.0.9+9"},
		map[string]string{"default-version": "21", "old.alias": "temurin@17"})

	logs := uninstallLogged(t, "temurin@21.0.5").String()
	if strings.Contains(logs, "now selects") {
		t.Fatalf("quiet logs = %q, want no info lines", logs)
	}
	logs = uninstallLogged(t, "temurin@17").String()
	if strings.Count(logs, "no longer matches an installed JDK") != 1 {
		t.Fatalf("quiet logs = %q, want the alias warning unchanged", logs)
	}
}

func TestUninstallReportsLegacyDefaultAliasSelectingAnotherJDK(t *testing.T) {
	setupUninstallPins(t, two21Builds(), map[string]string{"default.alias": "21"})

	logs := uninstallInfoLogged(t, "temurin@21.0.5")
	if !strings.Contains(logs, "default 21 now selects temurin@21.0.4") || strings.Count(logs, "default.alias") > 1 {
		t.Fatalf("logs = %q, want the default info line and at most one migration warning", logs)
	}
}
