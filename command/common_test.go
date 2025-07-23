package command

import (
	"github.com/felipebz/javm/discoapi"
	"strings"
	"testing"
)

func TestMakePackageIndex(t *testing.T) {
	mock := &mockPackagesClient{
		Pkgs: []discoapi.Package{
			{JavaVersion: "21.0.1+9", Distribution: "temurin", DistributionVersion: "21.0.1"},
			{JavaVersion: "17+35", Distribution: "zulu", DistributionVersion: "17"},
		},
	}
	idx, err := makePackageIndex(mock, "linux", "amd64", "")
	if err != nil {
		t.Fatal(err)
	}

	if !hasPackageWithVersion(idx, "temurin", "21.0.1") {
		t.Errorf("expected to find package temurin@21.0.1")
	}

	if !hasPackageWithVersion(idx, "zulu", "17") {
		t.Errorf("expected to find package zulu@17")
	}

	if len(idx.Sorted) != 2 {
		t.Errorf("expected 2 versions in Sorted")
	}
}

func hasPackageWithVersion(idx *packageIndex, distribution, version string) bool {
	for _, pkg := range idx.ByVersion {
		if pkg.Distribution == distribution && strings.HasPrefix(pkg.JavaVersion, version) {
			return true
		}
	}
	return false
}
