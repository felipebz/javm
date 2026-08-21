package command

import (
	"context"
	"fmt"
	"io"
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
	if before, _, ok := strings.Cut(javaVersion, "+"); ok {
		return before
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
		if err := os.WriteFile(fd3, []byte(strings.Join(out, "\n")), 0600); err != nil {
			return fmt.Errorf("write fd3 %q: %w", fd3, err)
		}
		return nil
	}

	fd := os.NewFile(3, "fd3")
	if fd == nil {
		return shellIntegrationUnavailable()
	}
	return writeShellEnvironment(fd, out)
}

func writeShellEnvironment(w io.Writer, out []string) error {
	for _, line := range out {
		if _, err := fmt.Fprintln(w, line); err != nil {
			return shellIntegrationUnavailable()
		}
	}
	return nil
}

func shellIntegrationUnavailable() error {
	return shellIntegrationError(detectedShellHint())
}

func shellIntegrationError(hint shellIntegrationHint, ok bool) error {
	if ok {
		return fmt.Errorf("%w; enable javm for %s with:\n%s", ErrShellIntegration, hint.name, hint.command)
	}
	return fmt.Errorf("%w; run `javm init <shell>` and invoke javm through the generated shell wrapper", ErrShellIntegration)
}
