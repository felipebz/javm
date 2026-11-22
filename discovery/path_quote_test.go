package discovery

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDiscoveryWarningShowsPathWithSingleBackslashes(t *testing.T) {
	cause := errors.New("boom")
	for _, tc := range []struct {
		warning *DiscoveryWarning
		want    string
	}{
		{&DiscoveryWarning{Source: "intellij", Path: `C:\Users\me\.jdks\x`, Err: cause},
			`discover JDKs from source "intellij" at path "C:\Users\me\.jdks\x": boom`},
		{&DiscoveryWarning{Location: `C:\Program Files\Java`, Err: cause},
			`discover JDKs at location "C:\Program Files\Java": boom`},
		{&DiscoveryWarning{Source: "system", Path: "C:\\jdk\x1b[2J\a", Err: cause},
			`discover JDKs from source "system" at path "C:\jdk[2J": boom`},
		// Source names keep %q quoting.
		{&DiscoveryWarning{Source: `in"tel`, Err: cause},
			`discover JDKs from source "in\"tel": boom`},
	} {
		if got := tc.warning.Error(); got != tc.want {
			t.Fatalf("Error() = %q, want %q", got, tc.want)
		}
	}
}

func TestCorruptCacheWarningShowsPathWithSingleBackslashes(t *testing.T) {
	cacheFile := filepath.Join(t.TempDir(), `javm\cache.json`)
	if err := os.MkdirAll(filepath.Dir(cacheFile), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cacheFile, []byte("garbage{{"), 0o600); err != nil {
		t.Fatal(err)
	}
	d := NewManager(cacheFile, time.Hour)
	var warnings []string
	d.Warn = func(err error) { warnings = append(warnings, err.Error()) }
	if _, err := d.DiscoverAll(context.Background()); err != nil {
		t.Fatal(err)
	}
	want := `ignoring corrupt discovery cache "` + cacheFile + `": `
	if len(warnings) != 1 || !strings.HasPrefix(warnings[0], want) {
		t.Fatalf("warnings = %q, want one starting with %s", warnings, want)
	}
}
