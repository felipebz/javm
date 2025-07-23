package command

import (
	"fmt"
	"github.com/felipebz/javm/discoapi"
	"github.com/felipebz/javm/semver"
	log "github.com/sirupsen/logrus"
	"sort"
	"strings"
)

type PackagesClient interface {
	GetPackages(os, arch, distribution, version string) ([]discoapi.Package, error)
}

type packageIndex struct {
	ByVersion map[*semver.Version]discoapi.Package
	Sorted    []*semver.Version
}

func makePackageIndex(client PackagesClient, osFlag, archFlag, distributionFlag string) (*packageIndex, error) {
	pkgs, err := client.GetPackages(osFlag, archFlag, distributionFlag, "")
	if err != nil {
		return nil, err
	}

	byVersion := make(map[*semver.Version]discoapi.Package)
	var sorted []*semver.Version

	for _, pkg := range pkgs {
		v, err := semver.ParseVersion(fmt.Sprintf("%s@%s", pkg.Distribution, stripBuildSuffix(pkg.JavaVersion)))
		if err == nil {
			byVersion[v] = pkg
			sorted = append(sorted, v)
		}
	}
	sort.Sort(semver.VersionSlice(sorted))
	return &packageIndex{
		ByVersion: byVersion,
		Sorted:    sorted,
	}, nil
}

func stripBuildSuffix(javaVersion string) string {
	if idx := strings.Index(javaVersion, "+"); idx != -1 {
		return javaVersion[:idx]
	}
	return javaVersion
}

func parseTrimTo(value string) semver.VersionPart {
	switch strings.ToLower(value) {
	case "major":
		return semver.VPMajor
	case "minor":
		return semver.VPMinor
	case "patch":
		return semver.VPPatch
	default:
		log.Fatal("Unexpected value of --latest (must be either \"major\", \"minor\" or \"patch\")")
		return -1
	}
}
