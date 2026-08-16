package command

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/felipebz/javm/discoapi"
	"github.com/felipebz/javm/semver"
)

type PackagesClient interface {
	GetPackagesContext(ctx context.Context, os, arch, distribution, version string) ([]discoapi.Package, error)
}

type PackagesWithInfoClient interface {
	PackagesClient
	GetPackageInfoContext(ctx context.Context, id string) (*discoapi.PackageInfo, error)
}

type packageIndex struct {
	ByVersion map[*semver.Version]discoapi.Package
	Sorted    []*semver.Version
}

func makePackageIndex(ctx context.Context, client PackagesClient, osFlag, archFlag, distributionFlag string) (*packageIndex, error) {
	pkgs, err := client.GetPackagesContext(ctx, osFlag, archFlag, distributionFlag, "")
	if err != nil {
		return nil, NetworkError(err)
	}
	return packageIndexFromPackages(pkgs), nil
}

func packageIndexFromPackages(pkgs []discoapi.Package) *packageIndex {
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
	return &packageIndex{ByVersion: byVersion, Sorted: sorted}
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
		return -1
	}
}

func printForShellToEval(out []string, fd3 string) error {
	if fd3 != "" {
		return os.WriteFile(fd3, []byte(strings.Join(out, "\n")), 0600)
	}

	fd := os.NewFile(3, "fd3")
	if fd == nil {
		return fmt.Errorf("shell integration is not active; run `javm init <shell>` first")
	}
	for _, line := range out {
		if _, err := fmt.Fprintln(fd, line); err != nil {
			return err
		}
	}
	return nil
}
