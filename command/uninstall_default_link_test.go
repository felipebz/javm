package command

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/felipebz/javm/cfg"
	"github.com/felipebz/javm/discovery"
)

// discoveredAndManaged21 uses the real discovery, with its cache, limited to
// javm and the intellij source under a scratch home. It installs a managed
// temurin@21.0.4, a discovered 21 where the intellij source scans and the
// given pin files, and returns the discovered JDK's path: the bundle root on
// macOS, as discovery reports it and javm links it.
func discoveredAndManaged21(t *testing.T, managed map[string]string, pins map[string]string) string {
	t.Helper()
	skipWithoutSymlinks(t)
	t.Setenv("JAVM_HOME", t.TempDir())
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	config := `{"enabled":true,"sources":{"system":false,"gradle":false,"jabba":false},"cache_ttl":3600000000000}`
	if err := os.MkdirAll(filepath.Join(cfg.Dir(), "autodiscover"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg.Dir(), "autodiscover", "config.json"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	for identifier, version := range managed {
		writeManagedJDK(t, identifier, version, "")
	}
	discovered := filepath.Join(home, ".jdks", "corretto-21.0.2")
	if runtime.GOOS == "darwin" {
		discovered = filepath.Join(home, "Library", "Java", "JavaVirtualMachines", "corretto-21.0.2.jdk")
	}
	jdkHome := filepath.FromSlash(discovery.ExpectedJDKDir(discovered, runtime.GOOS))
	java := filepath.FromSlash(discovery.ExpectedJavaPath(discovered, runtime.GOOS))
	if err := os.MkdirAll(filepath.Dir(java), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(java, []byte("java"), 0o755); err != nil {
		t.Fatal(err)
	}
	release := "JAVA_VERSION=\"21.0.2\"\nJAVA_VENDOR=\"Amazon.com Inc.\"\nOS_ARCH=\"x86_64\"\n"
	if err := os.WriteFile(filepath.Join(jdkHome, "release"), []byte(release), 0o644); err != nil {
		t.Fatal(err)
	}
	for file, content := range pins {
		if err := os.WriteFile(filepath.Join(cfg.Dir(), file), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return discovered
}

func assertLinkTarget(t *testing.T, name, want string) {
	t.Helper()
	link := filepath.Join(cfg.Dir(), "jdk", name)
	if _, err := os.Stat(link); err != nil {
		t.Fatalf("jdk/%s does not resolve: %v", name, err)
	}
	if got := canonicalHome(link); got != canonicalHome(want) {
		t.Fatalf("jdk/%s -> %s, want %s", name, got, want)
	}
}

func TestUninstallPointsDefaultAtDiscoveredJDK(t *testing.T) {
	discovered := discoveredAndManaged21(t, map[string]string{"temurin@21.0.4": "21.0.4+7"}, map[string]string{"default-version": "21"})

	logs := uninstallInfoLogged(t, "temurin@21.0.4")
	assertLinkTarget(t, "default", discovered)
	if !strings.Contains(logs, "default -> "+discovered) || strings.Contains(logs, "default -> "+filepath.Join(cfg.Dir(), "jdk", "temurin@21.0.4")) {
		t.Fatalf("logs = %q, want default linked to %s", logs, discovered)
	}
}

func TestUninstallPointsDefaultAndAliasAtDiscoveredJDK(t *testing.T) {
	discovered := discoveredAndManaged21(t, map[string]string{"temurin@21.0.4": "21.0.4+7"},
		map[string]string{"default-version": "21", "work.alias": "21"})

	uninstallInfoLogged(t, "temurin@21.0.4")
	assertLinkTarget(t, "default", discovered)
	assertLinkTarget(t, "work", discovered)
}

func TestUninstallPointsDefaultAtRemainingManagedBuild(t *testing.T) {
	discoveredAndManaged21(t, map[string]string{"temurin@21.0.4": "21.0.4+7", "temurin@21.0.5": "21.0.5+11"},
		map[string]string{"default-version": "21"})

	uninstallInfoLogged(t, "temurin@21.0.5")
	assertLinkTarget(t, "default", filepath.Join(cfg.Dir(), "jdk", "temurin@21.0.4"))
}

func TestUninstallWithUnclearableCacheWarnsOnce(t *testing.T) {
	discoveredAndManaged21(t, map[string]string{"temurin@21.0.4": "21.0.4+7"}, map[string]string{"default-version": "21"})
	breakCacheInvalidation(t)

	logs := uninstallInfoLogged(t, "temurin@21.0.4")
	if got := strings.Count(logs, cacheWarning); got != 1 {
		t.Fatalf("logs = %q, want one cache warning", logs)
	}
	if _, err := os.Stat(filepath.Join(cfg.Dir(), "jdk", "default")); err != nil {
		t.Fatalf("jdk/default does not resolve: %v", err)
	}
}
