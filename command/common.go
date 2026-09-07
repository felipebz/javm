package command

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/felipebz/javm/discoapi"
	"github.com/felipebz/javm/javaversion"
)

type PackagesClient interface {
	GetPackagesContext(ctx context.Context, os, arch, distribution, version string) ([]discoapi.Package, error)
}

type PackagesWithInfoClient interface {
	PackagesClient
	GetPackageInfoContext(ctx context.Context, id string) (*discoapi.PackageInfo, error)
}

type packageIndex struct {
	ByVersion map[*javaversion.Version]discoapi.Package
	Sorted    []*javaversion.Version
	invalid   []invalidPackageVersion
}

type invalidPackageVersion struct {
	packageValue discoapi.Package
	err          error
}

func makePackageIndex(ctx context.Context, client PackagesClient, osFlag, archFlag, distributionFlag string) (*packageIndex, error) {
	pkgs, err := client.GetPackagesContext(ctx, osFlag, archFlag, distributionFlag, "")
	if err != nil {
		return nil, NetworkError(err)
	}
	index := packageIndexFromPackages(pkgs)
	for _, invalid := range index.invalid {
		loggerFromContext(ctx).Debugf("ignoring package %q with invalid Java version %q: %v", invalid.packageValue.Id, invalid.packageValue.JavaVersion, invalid.err)
	}
	return index, nil
}

func packageIndexFromPackages(pkgs []discoapi.Package) *packageIndex {
	byVersion := make(map[*javaversion.Version]discoapi.Package)
	var sorted []*javaversion.Version
	seen := make(map[string]*javaversion.Version)
	sortedIndex := make(map[string]int)
	var invalid []invalidPackageVersion

	for _, pkg := range pkgs {
		v, err := javaversion.ParseVersion(fmt.Sprintf("%s@%s", pkg.Distribution, pkg.JavaVersion))
		if err != nil {
			invalid = append(invalid, invalidPackageVersion{packageValue: pkg, err: err})
			continue
		}
		key := v.Canonical()
		if previous, ok := seen[key]; ok {
			if packageSortKey(pkg) < packageSortKey(byVersion[previous]) {
				delete(byVersion, previous)
				seen[key] = v
				byVersion[v] = pkg
				sorted[sortedIndex[key]] = v
			}
			continue
		}
		seen[key] = v
		byVersion[v] = pkg
		sortedIndex[key] = len(sorted)
		sorted = append(sorted, v)
	}
	sort.Sort(javaversion.VersionSlice(sorted))
	return &packageIndex{ByVersion: byVersion, Sorted: sorted, invalid: invalid}
}

func packageSortKey(pkg discoapi.Package) string {
	return strings.Join([]string{pkg.Id, pkg.DistributionVersion, pkg.JavaVersion}, "\x00")
}

func parseTrimTo(value string) javaversion.VersionPart {
	switch strings.ToLower(value) {
	case "major":
		return javaversion.VPFeature
	case "minor":
		return javaversion.VPInterim
	case "patch":
		// Keep the existing three-component CLI grouping. In Java terms
		// this is the UPDATE component; VPJavaPatch is available to callers
		// that need the fourth component explicitly.
		return javaversion.VPUpdate
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
