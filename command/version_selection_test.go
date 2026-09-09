package command

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felipebz/javm/cfg"
	"github.com/felipebz/javm/discoapi"
	"github.com/felipebz/javm/discovery"
)

type versionSelectionClient struct {
	packages []discoapi.Package
	archive  string
	infoID   string
}

func (c *versionSelectionClient) GetPackagesContext(context.Context, string, string, string, string) ([]discoapi.Package, error) {
	return c.packages, nil
}

func (c *versionSelectionClient) GetPackageInfoContext(_ context.Context, id string) (*discoapi.PackageInfo, error) {
	c.infoID = id
	return &discoapi.PackageInfo{DirectDownloadUri: "file://" + filepath.ToSlash(c.archive)}, nil
}

func TestRunInstallSelectsLatestCompleteJavaVersion(t *testing.T) {
	archive := makeZipArchive(t, []zipTestEntry{{name: javaArchivePath(), body: "java", mode: 0755}})
	client := &versionSelectionClient{
		packages: []discoapi.Package{
			{Id: "baseline", JavaVersion: "25.0.4+7", Distribution: "temurin", DistributionVersion: "25.0.4"},
			{Id: "security", JavaVersion: "25.0.4.1+1", Distribution: "temurin", DistributionVersion: "25.0.4.1"},
		},
		archive: archive,
	}

	destination := filepath.Join(t.TempDir(), "jdk")
	version, err := runInstall(context.Background(), client, "temurin@25", destination)
	if err != nil {
		t.Fatalf("runInstall() error = %v", err)
	}
	if version != "temurin@25.0.4.1+1" {
		t.Fatalf("installed version = %q, want %q", version, "temurin@25.0.4.1+1")
	}
	if client.infoID != "security" {
		t.Fatalf("package info requested for %q, want security", client.infoID)
	}
}

func TestRunInstallSelectsHighestBuildForSameJavaVersion(t *testing.T) {
	archive := makeZipArchive(t, []zipTestEntry{{name: javaArchivePath(), body: "java", mode: 0755}})
	client := &versionSelectionClient{
		packages: []discoapi.Package{
			{Id: "build-6", JavaVersion: "25.0.4+6", Distribution: "temurin", DistributionVersion: "25.0.4"},
			{Id: "build-7", JavaVersion: "25.0.4+7", Distribution: "temurin", DistributionVersion: "25.0.4"},
		},
		archive: archive,
	}

	destination := filepath.Join(t.TempDir(), "jdk")
	version, err := runInstall(context.Background(), client, "temurin@25.0.4", destination)
	if err != nil {
		t.Fatalf("runInstall() error = %v", err)
	}
	if version != "temurin@25.0.4+7" {
		t.Fatalf("installed version = %q, want %q", version, "temurin@25.0.4+7")
	}
	if client.infoID != "build-7" {
		t.Fatalf("package info requested for %q, want build-7", client.infoID)
	}

	client.packages = []discoapi.Package{
		{Id: "build-7", JavaVersion: "25.0.4+7", Distribution: "temurin", DistributionVersion: "25.0.4"},
		{Id: "build-6", JavaVersion: "25.0.4+6", Distribution: "temurin", DistributionVersion: "25.0.4"},
	}
	client.infoID = ""
	reverseDestination := filepath.Join(t.TempDir(), "jdk")
	version, err = runInstall(context.Background(), client, "temurin@25.0.4", reverseDestination)
	if err != nil {
		t.Fatalf("reverse-order runInstall() error = %v", err)
	}
	if version != "temurin@25.0.4+7" || client.infoID != "build-7" {
		t.Fatalf("reverse-order selection chose version %q and package %q", version, client.infoID)
	}
}

func TestRunInstallResolvesExplicitJavaPatchSelector(t *testing.T) {
	archive := makeZipArchive(t, []zipTestEntry{{name: javaArchivePath(), body: "java", mode: 0755}})
	client := &versionSelectionClient{
		packages: []discoapi.Package{
			{Id: "baseline", JavaVersion: "25.0.4+7", Distribution: "temurin", DistributionVersion: "25.0.4"},
			{Id: "security", JavaVersion: "25.0.4.1+1", Distribution: "temurin", DistributionVersion: "25.0.4.1"},
		},
		archive: archive,
	}

	destination := filepath.Join(t.TempDir(), "jdk")
	version, err := runInstall(context.Background(), client, "temurin@25.0.4.1", destination)
	if err != nil {
		t.Fatalf("runInstall() error = %v", err)
	}
	if version != "temurin@25.0.4.1+1" || client.infoID != "security" {
		t.Fatalf("explicit selector chose version %q and package %q", version, client.infoID)
	}
}

func TestRunInstallUsesBuildFreeManagedDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("JAVM_HOME", home)
	archive := makeZipArchive(t, []zipTestEntry{
		{name: javaArchivePath(), body: "java", mode: 0755},
		{name: "jdk/release", body: "JAVA_VERSION=\"25.0.4.1\"\nJAVA_VENDOR=\"Eclipse Adoptium\"\nOS_ARCH=\"x86_64\"", mode: 0644},
	})
	client := &versionSelectionClient{
		packages: []discoapi.Package{{
			Id:                  "security",
			JavaVersion:         "25.0.4.1+1",
			Distribution:        "temurin",
			DistributionVersion: "25.0.4.1",
		}},
		archive: archive,
	}

	version, err := runInstall(context.Background(), client, "temurin@25", "")
	if err != nil {
		t.Fatalf("runInstall() error = %v", err)
	}
	if version != "temurin@25.0.4.1+1" {
		t.Fatalf("installed version = %q", version)
	}
	buildFreePath := filepath.Join(cfg.Dir(), "jdk", "temurin@25.0.4.1")
	if _, err := os.Stat(buildFreePath); err != nil {
		t.Fatalf("build-free managed directory is missing: %v", err)
	}
	fullVersionPath := filepath.Join(cfg.Dir(), "jdk", "temurin@25.0.4.1+1")
	if _, err := os.Stat(fullVersionPath); !os.IsNotExist(err) {
		t.Fatalf("full build version path unexpectedly exists: %v", err)
	}
	version, err = runInstall(context.Background(), client, "temurin@25", "")
	if err != nil {
		t.Fatalf("idempotent runInstall() error = %v", err)
	}
	if version != "temurin@25.0.4.1+1" {
		t.Fatalf("idempotent installed version = %q", version)
	}
}

func TestFindBestMatchJDKUsesCompleteJavaVersionOrdering(t *testing.T) {
	jdks := []discovery.JDK{
		{Identifier: "temurin@25.0.4+7", Version: "25.0.4+7", Source: "javm"},
		{Identifier: "temurin@25.0.4.1+1", Version: "25.0.4.1+1", Source: "javm"},
	}

	got, err := FindBestMatchJDK(jdks, "temurin@25")
	if err != nil {
		t.Fatalf("FindBestMatchJDK() error = %v", err)
	}
	if got.Identifier != "temurin@25.0.4.1+1" {
		t.Fatalf("resolved JDK = %q, want security release", got.Identifier)
	}
}

func TestFindBestMatchJDKUsesFullDiscoveredVersionWithQualifiedIdentifier(t *testing.T) {
	jdks := []discovery.JDK{{
		Identifier: "temurin@25",
		Version:    "25.0.4.1+1",
		Source:     "javm",
	}}

	got, err := FindBestMatchJDK(jdks, "temurin@25.0.4.1")
	if err != nil {
		t.Fatalf("FindBestMatchJDK() error = %v", err)
	}
	if got.Identifier != "temurin@25" {
		t.Fatalf("resolved JDK = %q, want temurin@25", got.Identifier)
	}
}

func TestUseResolvesBuildFreeManagedDirectoryByCompleteVersion(t *testing.T) {
	home := t.TempDir()
	t.Setenv("JAVM_HOME", home)
	t.Setenv("PATH", "/usr/bin")
	cleanup := setupMockLs()
	defer cleanup()

	path := filepath.Join(home, "jdk", "temurin@25.0.4.1")
	mockLsResult = []discovery.JDK{{
		Identifier: "temurin@25.0.4.1",
		Version:    "25.0.4.1+1",
		Source:     "javm",
		Path:       path,
	}}

	env, err := UseContext(context.Background(), "temurin@25.0.4.1+1")
	if err != nil {
		t.Fatalf("UseContext() error = %v", err)
	}
	if !strings.Contains(strings.Join(env, "\n"), "SET\tJAVA_HOME\t"+path) {
		t.Fatalf("use environment does not select build-free managed directory: %v", env)
	}
}
