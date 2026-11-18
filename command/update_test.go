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
