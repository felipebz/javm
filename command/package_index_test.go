package command

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/felipebz/javm/discoapi"
	"github.com/felipebz/javm/javaversion"
)

// loadRebuildPackages reads real DiscoAPI data (linux x64): Zulu October builds
// vs their +1 rebuilds, FX and non-FX, plus Liberica 25.0.5.
func loadRebuildPackages(t *testing.T) []discoapi.Package {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "discoapi_rebuild_packages.json"))
	if err != nil {
		t.Fatal(err)
	}
	var response discoapi.PackagesResponse
	if err := json.Unmarshal(data, &response); err != nil {
		t.Fatal(err)
	}
	return response.Packages
}

func latestPackage(t *testing.T, index *packageIndex, selector string) discoapi.Package {
	t.Helper()
	rng, err := javaversion.ParseRange(selector)
	if err != nil {
		t.Fatal(err)
	}
	version := index.latestMatching(rng)
	if version == nil {
		t.Fatalf("no package matches %s", selector)
	}
	return index.ByVersion[version]
}

func TestLatestMatchingPrefersZuluNonFXRebuild(t *testing.T) {
	packages := loadRebuildPackages(t)
	tests := []struct {
		selector string
		wantID   string
	}{
		{"zulu@25", "2f0ce3bb817fe13c1147130ac8b93c27"},
		{"zulu@25.0.5", "2f0ce3bb817fe13c1147130ac8b93c27"},
		{"zulu@21", "b7e165c2da16d93a21869a32e51c181f"},
		{"zulu@17", "4140bf5b93b15bd3a0c4bd50f03d4999"},
	}
	for _, order := range []string{"api", "reversed"} {
		input := slices.Clone(packages)
		if order == "reversed" {
			slices.Reverse(input)
		}
		index := packageIndexFromPackages(input)
		for _, tt := range tests {
			t.Run(order+"/"+tt.selector, func(t *testing.T) {
				got := latestPackage(t, index, tt.selector)
				if got.Id != tt.wantID || got.JavaFXBundled {
					t.Fatalf("selected %s %s (%s, fx=%v), want non-FX rebuild %s", got.JavaVersion, got.DistributionVersion, got.Id, got.JavaFXBundled, tt.wantID)
				}
			})
		}
	}
}

func TestLatestMatchingKeepsLibericaBuildOrdering(t *testing.T) {
	index := packageIndexFromPackages(loadRebuildPackages(t))
	got := latestPackage(t, index, "liberica@25")
	if got.JavaVersion != "25.0.5+11" {
		t.Fatalf("selected liberica %s (%s), want 25.0.5+11", got.JavaVersion, got.DistributionVersion)
	}
}

func TestLatestMatchingKeepsTemurinBuildOrdering(t *testing.T) {
	index := packageIndexFromPackages([]discoapi.Package{
		{Id: "new", Distribution: "temurin", JavaVersion: "21.0.5+11", DistributionVersion: "21.0.5"},
		{Id: "old", Distribution: "temurin", JavaVersion: "21.0.5+1", DistributionVersion: "21.0.5.1"},
	})
	if got := latestPackage(t, index, "temurin@21"); got.Id != "new" {
		t.Fatalf("selected temurin %s, want 21.0.5+11", got.JavaVersion)
	}
}

func TestTrimToShowsZuluRebuild(t *testing.T) {
	index := packageIndexFromPackages(loadRebuildPackages(t))
	var got []string
	for _, version := range index.trimTo(index.Sorted, javaversion.VPUpdate) {
		if version.Qualifier() == "zulu" {
			got = append(got, index.ByVersion[version].JavaVersion)
		}
	}
	want := []string{"17.0.21+1", "21.0.13+1", "25.0.4+7", "25.0.5+1"}
	if !slices.Equal(got, want) {
		t.Fatalf("trimmed zulu versions = %v, want %v", got, want)
	}
}

func TestCompareDistributionVersions(t *testing.T) {
	tests := []struct {
		left, right string
		want        int
	}{
		{"25.38.109", "25.38.107", 1},
		{"25.38.9", "25.38.10", -1},
		{"25.38", "25.38.0", 0},
		{"25.38.1", "25.38", 1},
		{"25.0.5", "not-a-version", 0},
		{"", "25.38.107", 0},
	}
	for _, tt := range tests {
		if got := compareDistributionVersions(tt.left, tt.right); got != tt.want {
			t.Errorf("compareDistributionVersions(%q, %q) = %d, want %d", tt.left, tt.right, got, tt.want)
		}
	}
}
