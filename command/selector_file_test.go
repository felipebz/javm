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
	"github.com/felipebz/javm/discovery"
)

func TestReadDefaultVersionAcceptsTrailingNewline(t *testing.T) {
	for _, content := range []string{"21\n", "21\r\n", "21 \n\n", "  21\t\n"} {
		t.Setenv("JAVM_HOME", t.TempDir())
		if err := os.WriteFile(defaultVersionPath(), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		if got, err := readDefaultVersion(); err != nil || got != "21" {
			t.Errorf("readDefaultVersion() with %q = %q, %v, want 21", content, got, err)
		}
	}
}

func TestReadDefaultVersionRejectsEmbeddedControlCharacters(t *testing.T) {
	for _, content := range []string{"17\necho compromised", "17\n21\n", "17\r21", "\n21", "21\x00", "21\x00\n"} {
		t.Setenv("JAVM_HOME", t.TempDir())
		if err := os.WriteFile(defaultVersionPath(), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		if got, err := readDefaultVersion(); !errors.Is(err, ErrUsage) {
			t.Errorf("readDefaultVersion() with %q = %q, %v, want a usage error", content, got, err)
		}
	}
}

func writeAliasFile(t *testing.T, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(cfg.Dir(), name+".alias"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestAliasFileWithTrailingNewlineKeepsItsLink(t *testing.T) {
	t.Setenv("JAVM_HOME", t.TempDir())
	writeManagedJDK(t, "temurin@21.0.4", "21.0.4+7", "jdk")
	jdks, err := discovery.NewJavmSource().DiscoverManaged(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	stubDiscovery(t, jdks...)
	writeAliasFile(t, "work", "temurin@21\r\n")

	if got := getAlias("work"); got != "temurin@21" {
		t.Fatalf("getAlias(work) = %q, want temurin@21", got)
	}
	if err := refreshAliasLinks(context.Background()); err != nil {
		t.Fatalf("refreshAliasLinks() error = %v", err)
	}
	if got := aliasLinkTarget(t, "work"); got != jdks[0].Path {
		t.Fatalf("jdk/work -> %q, want %q", got, jdks[0].Path)
	}
}

func TestInvalidAliasFileIsNotSet(t *testing.T) {
	t.Setenv("JAVM_HOME", t.TempDir())
	writeAliasFile(t, "work", "temurin@21\necho compromised")

	if got := getAlias("work"); got != "" {
		t.Fatalf("getAlias(work) = %q, want an invalid alias to read as not set", got)
	}
	err := runAlias(t, "work")
	if !errors.Is(err, ErrUsage) || !strings.Contains(err.Error(), `alias "work" is invalid`) {
		t.Fatalf("alias work error = %v, want a usage error naming the invalid alias", err)
	}
	if strings.Contains(err.Error(), cfg.Dir()) {
		t.Fatalf("alias work error = %q, want no file path", err)
	}
}

// warningsNaming counts the logged warnings that mention name.
func warningsNaming(logs *bytes.Buffer, name string) int {
	count := 0
	for line := range strings.Lines(logs.String()) {
		if strings.Contains(line, "level=warning") && strings.Contains(line, name) {
			count++
		}
	}
	return count
}

// An unreadable pin may have been meant to hold the old build, so update
// installs the new build but removes nothing.
func TestUpdateKeepsOldBuildsWhenPinFileIsInvalid(t *testing.T) {
	tests := []struct {
		name  string
		write func(t *testing.T)
		names string
	}{
		{name: "alias", names: `alias \"bad\"`, write: func(t *testing.T) {
			writeAliasFile(t, "bad", "temurin@25.0.4\necho compromised")
		}},
		{name: "default-version", names: "default-version", write: func(t *testing.T) {
			if err := os.WriteFile(defaultVersionPath(), []byte("temurin@25.0.4\necho compromised"), 0o600); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := temurinUpdateFixture(t)
			originalLs := lsFunc
			lsFunc = func(ctx context.Context) ([]discovery.JDK, error) {
				return discovery.NewJavmSource().DiscoverManaged(ctx)
			}
			t.Cleanup(func() { lsFunc = originalLs })
			tt.write(t)
			ctx, logs := contextWithLogBuffer(context.Background())

			if err := runUpdate(ctx, client, &bytes.Buffer{}, "temurin@25", updateOptions{}); err != nil {
				t.Fatalf("runUpdate() error = %v", err)
			}
			assertExists(t, "temurin@25.0.5", true)
			assertExists(t, "temurin@25.0.4", true)
			if got := warningsNaming(logs, tt.names); got != 1 {
				t.Fatalf("logs = %q, want one warning naming %s", logs.String(), tt.names)
			}
			if strings.Contains(logs.String(), "usage error") {
				t.Fatalf("logs = %q, want no usage error marker", logs.String())
			}
		})
	}
}
