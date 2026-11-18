package command

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/felipebz/javm/cfg"
	"github.com/felipebz/javm/discoapi"
	"github.com/felipebz/javm/discovery"
	"github.com/spf13/cobra"
)

type updateClient struct {
	packages []discoapi.Package
	archives map[string]string
	infoIDs  []string
}

func (c *updateClient) GetPackagesContext(context.Context, string, string, string, string) ([]discoapi.Package, error) {
	return c.packages, nil
}

func (c *updateClient) GetPackageInfoContext(_ context.Context, id string) (*discoapi.PackageInfo, error) {
	c.infoIDs = append(c.infoIDs, id)
	return &discoapi.PackageInfo{DirectDownloadUri: "file://" + filepath.ToSlash(c.archives[id])}, nil
}

func testRelease(semanticVersion string) string {
	return fmt.Sprintf("SEMANTIC_VERSION=%q\nJAVA_VENDOR=\"Test\"\nOS_ARCH=\"x86_64\"", semanticVersion)
}

func writeManagedJDK(t *testing.T, identifier, semanticVersion, marker string) {
	t.Helper()
	home := filepath.FromSlash(discovery.ExpectedJDKDir(filepath.Join(cfg.Dir(), "jdk", identifier), runtime.GOOS))
	java := filepath.FromSlash(discovery.ExpectedJavaPath(filepath.Join(cfg.Dir(), "jdk", identifier), runtime.GOOS))
	if err := os.MkdirAll(filepath.Dir(java), 0o755); err != nil {
		t.Fatal(err)
	}
	for path, body := range map[string]string{
		java:                           "java",
		filepath.Join(home, "release"): testRelease(semanticVersion),
		filepath.Join(home, "marker"):  marker,
	} {
		if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

func updateArchive(t *testing.T, semanticVersion, marker string) string {
	t.Helper()
	return makeZipArchive(t, []zipTestEntry{
		{name: javaArchivePath(), body: "java", mode: 0o755},
		{name: "jdk/release", body: testRelease(semanticVersion), mode: 0o644},
		{name: "jdk/marker", body: marker, mode: 0o644},
	})
}

func managedJDKFile(identifier, name string) string {
	return filepath.Join(filepath.FromSlash(discovery.ExpectedJDKDir(filepath.Join(cfg.Dir(), "jdk", identifier), runtime.GOOS)), name)
}

func assertExists(t *testing.T, identifier string, want bool) {
	t.Helper()
	_, err := os.Lstat(filepath.Join(cfg.Dir(), "jdk", identifier))
	if want && err != nil {
		t.Fatalf("%s should exist: %v", identifier, err)
	}
	if !want && !os.IsNotExist(err) {
		t.Fatalf("%s should not exist: %v", identifier, err)
	}
}

func temurinUpdateFixture(t *testing.T) *updateClient {
	t.Helper()
	t.Setenv("JAVM_HOME", t.TempDir())
	writeManagedJDK(t, "temurin@25.0.4", "25.0.4+7", "old")
	return &updateClient{
		packages: []discoapi.Package{
			{Id: "old", JavaVersion: "25.0.4+7", Distribution: "temurin", DistributionVersion: "25.0.4"},
			{Id: "new", JavaVersion: "25.0.5+8", Distribution: "temurin", DistributionVersion: "25.0.5"},
		},
		archives: map[string]string{"new": updateArchive(t, "25.0.5+8", "new")},
	}
}

func TestUpdateReplacesZuluRebuildInPlace(t *testing.T) {
	t.Setenv("JAVM_HOME", t.TempDir())
	writeManagedJDK(t, "zulu@25.0.5", "25.0.5+8", "old")
	client := &updateClient{
		packages: []discoapi.Package{
			{Id: "original", JavaVersion: "25.0.5+8", Distribution: "zulu", DistributionVersion: "25.38.107"},
			{Id: "rebuild-fx", JavaVersion: "25.0.5+1", Distribution: "zulu", DistributionVersion: "25.38.109", JavaFXBundled: true},
			{Id: "rebuild", JavaVersion: "25.0.5+1", Distribution: "zulu", DistributionVersion: "25.38.109"},
		},
		archives: map[string]string{"rebuild": updateArchive(t, "25.0.5+1", "new")},
	}

	var out bytes.Buffer
	if err := runUpdate(context.Background(), client, &out, "zulu@25", updateOptions{}); err != nil {
		t.Fatalf("runUpdate() error = %v", err)
	}
	if !slices.Equal(client.infoIDs, []string{"rebuild"}) {
		t.Fatalf("package info requested for %v, want [rebuild]", client.infoIDs)
	}
	if got, _ := os.ReadFile(managedJDKFile("zulu@25.0.5", "marker")); string(got) != "new" {
		t.Fatalf("marker = %q, want new", got)
	}
	if got, _ := os.ReadFile(managedJDKFile("zulu@25.0.5", "release")); !strings.Contains(string(got), "25.0.5+1") {
		t.Fatalf("release = %q, want 25.0.5+1", got)
	}
	if !strings.Contains(out.String(), "zulu@25: zulu@25.0.5+8 -> zulu@25.0.5+1") {
		t.Fatalf("output = %q", out.String())
	}

	out.Reset()
	if err := runUpdate(context.Background(), client, &out, "zulu@25", updateOptions{}); err != nil {
		t.Fatalf("second runUpdate() error = %v", err)
	}
	if !strings.Contains(out.String(), "zulu@25: up to date (zulu@25.0.5+1)") {
		t.Fatalf("second output = %q", out.String())
	}
	if len(client.infoIDs) != 1 {
		t.Fatalf("second run requested package info: %v", client.infoIDs)
	}
}

func TestUpdateRemovesPreviousBuild(t *testing.T) {
	client := temurinUpdateFixture(t)
	if err := runUpdate(context.Background(), client, &bytes.Buffer{}, "temurin@25", updateOptions{}); err != nil {
		t.Fatalf("runUpdate() error = %v", err)
	}
	assertExists(t, "temurin@25.0.5", true)
	assertExists(t, "temurin@25.0.4", false)
	target, err := os.Readlink(filepath.Join(cfg.Dir(), "jdk", "temurin@25.0"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(target, "temurin@25.0.5") {
		t.Fatalf("temurin@25.0 -> %q, want temurin@25.0.5", target)
	}
}

func TestUpdateKeepPreservesPreviousBuild(t *testing.T) {
	client := temurinUpdateFixture(t)
	if err := runUpdate(context.Background(), client, &bytes.Buffer{}, "temurin@25", updateOptions{keep: true}); err != nil {
		t.Fatalf("runUpdate() error = %v", err)
	}
	assertExists(t, "temurin@25.0.5", true)
	assertExists(t, "temurin@25.0.4", true)
}

func TestUpdateKeepsBuildHeldByPinnedAlias(t *testing.T) {
	client := temurinUpdateFixture(t)
	originalLs := lsFunc
	lsFunc = func(ctx context.Context) ([]discovery.JDK, error) {
		return discovery.NewJavmSource().DiscoverManaged(ctx)
	}
	t.Cleanup(func() { lsFunc = originalLs })
	if err := setAlias("meu", "temurin@25.0.4"); err != nil {
		t.Fatal(err)
	}

	if err := runUpdate(context.Background(), client, &bytes.Buffer{}, "temurin@25", updateOptions{}); err != nil {
		t.Fatalf("runUpdate() error = %v", err)
	}
	assertExists(t, "temurin@25.0.5", true)
	assertExists(t, "temurin@25.0.4", true)
}

func TestUpdateDryRunChangesNothing(t *testing.T) {
	client := temurinUpdateFixture(t)
	var out bytes.Buffer
	if err := runUpdate(context.Background(), client, &out, "temurin@25", updateOptions{dryRun: true}); err != nil {
		t.Fatalf("runUpdate() error = %v", err)
	}
	if !strings.Contains(out.String(), "temurin@25: temurin@25.0.4+7 -> temurin@25.0.5+8") {
		t.Fatalf("output = %q", out.String())
	}
	if len(client.infoIDs) != 0 {
		t.Fatalf("dry run requested package info: %v", client.infoIDs)
	}
	assertExists(t, "temurin@25.0.5", false)
	assertExists(t, "temurin@25.0.4", true)
}

func TestUpdateAllInstalledLines(t *testing.T) {
	client := temurinUpdateFixture(t)
	writeManagedJDK(t, "temurin@21.0.10", "21.0.10+7", "old")
	client.packages = append(client.packages, discoapi.Package{Id: "21", JavaVersion: "21.0.10+7", Distribution: "temurin", DistributionVersion: "21.0.10"})

	var out bytes.Buffer
	if err := runUpdate(context.Background(), client, &out, "", updateOptions{}); err != nil {
		t.Fatalf("runUpdate() error = %v", err)
	}
	for _, want := range []string{
		"temurin@21: up to date (temurin@21.0.10+7)",
		"temurin@25: temurin@25.0.4+7 -> temurin@25.0.5+8",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output %q does not contain %q", out.String(), want)
		}
	}
}

func TestUpdateRequiresInstalledJDK(t *testing.T) {
	t.Setenv("JAVM_HOME", t.TempDir())
	err := runUpdate(context.Background(), &updateClient{}, &bytes.Buffer{}, "zulu@21", updateOptions{})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("runUpdate() error = %v, want ErrNotFound", err)
	}
}

func runUpdateCommand(t *testing.T, client *updateClient, args ...string) string {
	t.Helper()
	root := &cobra.Command{Use: "javm"}
	root.AddCommand(NewUpdateCommand(client))
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(append([]string{"update"}, args...))
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("update %v: %v\n%s", args, err, out.String())
	}
	return out.String()
}

func TestUpdateCommandAppliesFlags(t *testing.T) {
	t.Run("dry-run", func(t *testing.T) {
		client := temurinUpdateFixture(t)
		out := runUpdateCommand(t, client, "--dry-run", "temurin@25")
		if !strings.Contains(out, "temurin@25: temurin@25.0.4+7 -> temurin@25.0.5+8") || len(client.infoIDs) != 0 {
			t.Fatalf("--dry-run installed or printed wrong status: infoIDs=%v output=%q", client.infoIDs, out)
		}
		assertExists(t, "temurin@25.0.5", false)
	})
	t.Run("keep", func(t *testing.T) {
		client := temurinUpdateFixture(t)
		runUpdateCommand(t, client, "--keep", "temurin@25")
		assertExists(t, "temurin@25.0.5", true)
		assertExists(t, "temurin@25.0.4", true)
	})
}

func TestUpdateResolvesSelectorsAgainstInstalledJDKs(t *testing.T) {
	tests := []struct {
		name     string
		selector string
		alias    string
		want     string
	}{
		// Without a distribution the installed one is updated, so the newer
		// zulu@25 package must not be picked.
		{name: "unqualified", selector: "25", want: "25: temurin@25.0.4+7 -> temurin@25.0.5+8"},
		{name: "alias", selector: "work", alias: "temurin@25", want: "temurin@25: temurin@25.0.4+7 -> temurin@25.0.5+8"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := temurinUpdateFixture(t)
			client.packages = append(client.packages, discoapi.Package{Id: "zulu", JavaVersion: "25.0.9+1", Distribution: "zulu", DistributionVersion: "25.40.1"})
			if tt.alias != "" {
				if err := setAlias(tt.selector, tt.alias); err != nil {
					t.Fatal(err)
				}
			}
			var out bytes.Buffer
			if err := runUpdate(context.Background(), client, &out, tt.selector, updateOptions{dryRun: true}); err != nil {
				t.Fatalf("runUpdate() error = %v", err)
			}
			if !strings.Contains(out.String(), tt.want) {
				t.Fatalf("output = %q, want %q", out.String(), tt.want)
			}
		})
	}
}

// DiscoAPI drops old builds, so the installed package is usually missing from
// the remote list and the release and build decide instead.
func TestUpdateComparesBuildsMissingFromDiscoAPI(t *testing.T) {
	tests := []struct {
		name       string
		identifier string
		installed  string
		want       string
	}{
		{name: "older release", identifier: "temurin@25.0.4", installed: "25.0.4+7", want: "temurin@25: temurin@25.0.4+7 -> temurin@25.0.5+8"},
		{name: "older build", identifier: "temurin@25.0.5", installed: "25.0.5+7", want: "temurin@25: temurin@25.0.5+7 -> temurin@25.0.5+8"},
		{name: "newer release", identifier: "temurin@25.0.6", installed: "25.0.6+2", want: "temurin@25: up to date (temurin@25.0.6+2)"},
		{name: "release without build", identifier: "temurin@25.0.5", installed: "25.0.5", want: "temurin@25: up to date (temurin@25.0.5)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("JAVM_HOME", t.TempDir())
			writeManagedJDK(t, tt.identifier, tt.installed, "old")
			client := &updateClient{packages: []discoapi.Package{
				{Id: "new", JavaVersion: "25.0.5+8", Distribution: "temurin", DistributionVersion: "25.0.5"},
			}}
			var out bytes.Buffer
			if err := runUpdate(context.Background(), client, &out, "temurin@25", updateOptions{dryRun: true}); err != nil {
				t.Fatalf("runUpdate() error = %v", err)
			}
			if !strings.Contains(out.String(), tt.want) {
				t.Fatalf("output = %q, want %q", out.String(), tt.want)
			}
		})
	}
}

func TestUpdateAllContinuesPastLineWithoutRemotePackages(t *testing.T) {
	client := temurinUpdateFixture(t)
	writeManagedJDK(t, "zulu@21.0.1", "21.0.1+12", "old")

	var out bytes.Buffer
	if err := runUpdate(context.Background(), client, &out, "", updateOptions{dryRun: true}); err != nil {
		t.Fatalf("runUpdate() error = %v", err)
	}
	if !strings.Contains(out.String(), "temurin@25: temurin@25.0.4+7 -> temurin@25.0.5+8") {
		t.Fatalf("output = %q, want the temurin@25 update after skipping zulu@21", out.String())
	}
}

func TestUpdateKeepsBuildHeldByDefaultVersion(t *testing.T) {
	client := temurinUpdateFixture(t)
	if err := SetDefaultVersion("temurin@25.0.4"); err != nil {
		t.Fatal(err)
	}
	if err := runUpdate(context.Background(), client, &bytes.Buffer{}, "temurin@25", updateOptions{}); err != nil {
		t.Fatalf("runUpdate() error = %v", err)
	}
	assertExists(t, "temurin@25.0.5", true)
	assertExists(t, "temurin@25.0.4", true)
}

func TestUpdateWithoutManagedJDKs(t *testing.T) {
	t.Setenv("JAVM_HOME", t.TempDir())
	var out bytes.Buffer
	if err := runUpdate(context.Background(), &updateClient{}, &out, "", updateOptions{}); err != nil {
		t.Fatalf("runUpdate() error = %v", err)
	}
	if out.String() != "no managed JDKs to update\n" {
		t.Fatalf("output = %q", out.String())
	}
}

func TestRemoveManagedJDKRefusesSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks requires privileges on Windows")
	}
	t.Setenv("JAVM_HOME", t.TempDir())
	target := t.TempDir()
	if err := os.MkdirAll(filepath.Join(cfg.Dir(), "jdk"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(cfg.Dir(), "jdk", "system@21")); err != nil {
		t.Fatal(err)
	}
	if err := removeManagedJDK("system@21"); err == nil {
		t.Fatal("removeManagedJDK removed a symlinked JDK")
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("symlink target was touched: %v", err)
	}
}
