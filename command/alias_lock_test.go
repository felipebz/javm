package command

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/felipebz/javm/cfg"
)

func TestAliasLocksStayInLocksDirectory(t *testing.T) {
	t.Setenv("JAVM_HOME", t.TempDir())
	managedDiscoveryOnly(t)
	writeManagedJDK(t, "temurin@21.0.4", "21.0.4+7", "")
	failAllSymlinks(t)

	_ = runAlias(t, "work", "21")
	unalias := NewUnaliasCommand()
	unalias.SetOut(&bytes.Buffer{})
	unalias.SetErr(&bytes.Buffer{})
	unalias.SetArgs([]string{"work"})
	if err := unalias.Execute(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(cfg.Dir(), ".work.alias.lock")); !os.IsNotExist(err) {
		t.Fatalf(".work.alias.lock is in JAVM_HOME: %v", err)
	}
	if _, err := os.Stat(filepath.Join(cfg.Dir(), ".locks", "work.alias.lock")); err != nil {
		t.Fatalf("alias lock is not in .locks: %v", err)
	}
}
