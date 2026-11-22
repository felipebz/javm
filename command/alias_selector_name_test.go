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
)

const hidesSelector = "hides the version selector"

func TestAliasRejectsSelectorLikeNames(t *testing.T) {
	t.Setenv("JAVM_HOME", t.TempDir())
	managedDiscoveryOnly(t)
	writeManagedJDK(t, "temurin@21.0.4", "21.0.4+7", "")

	for name, selector := range map[string]string{"17": "temurin@21", "21.0": "21", "21.0.4": "21"} {
		err := runAlias(t, name, selector)
		if !errors.Is(err, ErrUsage) || !strings.Contains(err.Error(), `"`+name+`"`) || !strings.Contains(err.Error(), "version selector") {
			t.Errorf("alias %s %s error = %v, want a usage error naming the alias", name, selector, err)
		}
		if _, err := os.Stat(filepath.Join(cfg.Dir(), name+".alias")); !os.IsNotExist(err) {
			t.Errorf("%s.alias was written: %v", name, err)
		}
	}
}

func TestAliasAcceptsOrdinaryNames(t *testing.T) {
	t.Setenv("JAVM_HOME", t.TempDir())
	managedDiscoveryOnly(t)
	writeManagedJDK(t, "temurin@21.0.4", "21.0.4+7", "")
	writeManagedJDK(t, "temurin@17.0.12", "17.0.12+7", "")

	for name, selector := range map[string]string{"work": "21", "java17": "17", "lts": "21", "temurin": "21"} {
		if err := runAlias(t, name, selector); err != nil {
			t.Errorf("alias %s %s error = %v", name, selector, err)
		}
	}
}

func TestLegacySelectorLikeAliasStillResolves(t *testing.T) {
	t.Setenv("JAVM_HOME", t.TempDir())
	managedDiscoveryOnly(t)
	writeManagedJDK(t, "temurin@21.0.4", "21.0.4+7", "")
	writeManagedJDK(t, "temurin@17.0.12", "17.0.12+7", "")
	if err := os.WriteFile(filepath.Join(cfg.Dir(), "17.alias"), []byte("temurin@21"), 0o600); err != nil {
		t.Fatal(err)
	}

	ctx, logs := contextWithLogBuffer(context.Background())
	which := NewWhichCommand()
	var out bytes.Buffer
	which.SetOut(&out)
	which.SetArgs([]string{"17"})
	if err := which.ExecuteContext(ctx); err != nil || !strings.Contains(out.String(), "temurin@21.0.4") {
		t.Fatalf("which 17 = %q, %v, want the JDK of the legacy alias", out.String(), err)
	}
	if got := countLines(logs, hidesSelector+" 17"); got != 1 || !strings.Contains(logs.String(), "javm unalias 17") {
		t.Fatalf("logs = %q, want one warning with the rename hint", logs.String())
	}

	show := NewAliasCommand()
	var shown bytes.Buffer
	show.SetOut(&shown)
	show.SetArgs([]string{"17"})
	if err := show.Execute(); err != nil || shown.String() != "temurin@21\n" {
		t.Fatalf("alias 17 = %q, %v", shown.String(), err)
	}
	unalias := NewUnaliasCommand()
	unalias.SetOut(&bytes.Buffer{})
	unalias.SetArgs([]string{"17"})
	if err := unalias.Execute(); err != nil {
		t.Fatalf("unalias 17 error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(cfg.Dir(), "17.alias")); !os.IsNotExist(err) {
		t.Fatalf("17.alias still exists: %v", err)
	}
}

func TestUseDefaultIgnoresUnrelatedSelectorLikeAlias(t *testing.T) {
	t.Setenv("JAVM_HOME", t.TempDir())
	managedDiscoveryOnly(t)
	writeManagedJDK(t, "temurin@21.0.4", "21.0.4+7", "")
	if err := SetDefaultVersion("21"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg.Dir(), "17.alias"), []byte("temurin@21"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, logs := contextWithLogBuffer(context.Background())

	if records := useDefaultRecords(t, ctx); !strings.Contains(records, "temurin@21.0.4") || logs.Len() != 0 {
		t.Fatalf("records = %q, logs = %q, want the default and no warning", records, logs.String())
	}
}

func TestUpdateKeepsBuildHeldBySelectorLikeAlias(t *testing.T) {
	client := temurinUpdateFixture(t)
	managedDiscoveryOnly(t)
	if err := os.WriteFile(filepath.Join(cfg.Dir(), "25.0.alias"), []byte("temurin@25.0.4"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, logs := contextWithLogBuffer(context.Background())

	if err := runUpdate(ctx, client, &bytes.Buffer{}, "", updateOptions{}); err != nil {
		t.Fatalf("runUpdate() error = %v", err)
	}
	assertExists(t, "temurin@25.0.5", true)
	assertExists(t, "temurin@25.0.4", true)
	if got := countLines(logs, hidesSelector); got > 1 {
		t.Fatalf("logs = %q, want the warning at most once", logs.String())
	}
}
