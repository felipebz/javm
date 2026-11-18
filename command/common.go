package command

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"strconv"
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
			if preferPackage(pkg, byVersion[previous]) {
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
	index := &packageIndex{ByVersion: byVersion, Sorted: sorted, invalid: invalid}
	index.sort(index.Sorted)
	return index
}

func packageSortKey(pkg discoapi.Package) string {
	return strings.Join([]string{pkg.Id, pkg.DistributionVersion, pkg.JavaVersion}, "\x00")
}

// orderedByDistributionVersion lists distributions where distribution_version,
// not the build, orders rebuilds: Zulu 25.0.5+1 (25.38.109) supersedes +8 (25.38.107).
var orderedByDistributionVersion = map[string]bool{"zulu": true}

// preferPackage picks one package per canonical version. javm has no FX
// selector, so Zulu prefers the non-FX archive.
func preferPackage(candidate, current discoapi.Package) bool {
	if orderedByDistributionVersion[candidate.Distribution] && candidate.JavaFXBundled != current.JavaFXBundled {
		return !candidate.JavaFXBundled
	}
	return packageSortKey(candidate) < packageSortKey(current)
}

// compare is Java version order, with distribution_version breaking ties
// between builds of the same release.
func (idx *packageIndex) compare(left, right *javaversion.Version) int {
	if left.SameRelease(right) && orderedByDistributionVersion[left.Qualifier()] {
		if result := compareDistributionVersions(idx.ByVersion[left].DistributionVersion, idx.ByVersion[right].DistributionVersion); result != 0 {
			return result
		}
	}
	if result := left.Compare(right); result != 0 {
		return result
	}
	return strings.Compare(left.String(), right.String())
}

func (idx *packageIndex) sort(versions []*javaversion.Version) {
	slices.SortStableFunc(versions, idx.compare)
}

// latestMatching returns the newest indexed version inside rng, or nil.
func (idx *packageIndex) latestMatching(rng *javaversion.Range) *javaversion.Version {
	for _, v := range slices.Backward(idx.Sorted) {
		if rng.Contains(v) {
			return v
		}
	}
	return nil
}

// trimTo keeps the newest indexed version per group of the requested part.
func (idx *packageIndex) trimTo(versions []*javaversion.Version, part javaversion.VersionPart) []*javaversion.Version {
	latest := make(map[string]*javaversion.Version)
	for _, version := range versions {
		key := version.TrimTo(part)
		if previous, ok := latest[key]; !ok || idx.compare(version, previous) > 0 {
			latest[key] = version
		}
	}
	result := slices.Collect(maps.Values(latest))
	idx.sort(result)
	return result
}

// compareDistributionVersions compares dotted numbers segment by segment;
// non-numeric values compare equal.
func compareDistributionVersions(left, right string) int {
	leftParts, leftOK := parseDottedNumbers(left)
	rightParts, rightOK := parseDottedNumbers(right)
	if !leftOK || !rightOK {
		return 0
	}
	for i := range max(len(leftParts), len(rightParts)) {
		var l, r uint64
		if i < len(leftParts) {
			l = leftParts[i]
		}
		if i < len(rightParts) {
			r = rightParts[i]
		}
		if result := cmp.Compare(l, r); result != 0 {
			return result
		}
	}
	return 0
}

func parseDottedNumbers(value string) ([]uint64, bool) {
	if value == "" {
		return nil, false
	}
	fields := strings.Split(value, ".")
	parts := make([]uint64, len(fields))
	for i, field := range fields {
		part, err := strconv.ParseUint(field, 10, 64)
		if err != nil {
			return nil, false
		}
		parts[i] = part
	}
	return parts, true
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
		if err := validateShellEnvironmentRecords(out); err != nil {
			return err
		}
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
	if err := validateShellEnvironmentRecords(out); err != nil {
		return err
	}
	for _, line := range out {
		if _, err := fmt.Fprintln(w, line); err != nil {
			return shellIntegrationUnavailable()
		}
	}
	return nil
}

func validateShellEnvironmentRecords(out []string) error {
	for i, line := range out {
		if strings.ContainsAny(line, "\r\n\x00") {
			return fmt.Errorf("invalid shell environment record %d: CR, LF and NUL are not allowed", i+1)
		}
		operation, fields, ok := strings.Cut(line, "\t")
		if !ok {
			return fmt.Errorf("invalid shell environment record %d: missing tab-separated fields", i+1)
		}
		var key string
		switch operation {
		case "SET":
			var value string
			key, value, ok = strings.Cut(fields, "\t")
			if !ok || strings.Contains(value, "\t") {
				return fmt.Errorf("invalid shell environment record %d: SET requires exactly 3 tab-separated fields", i+1)
			}
		case "UNSET":
			key = fields
			if strings.Contains(key, "\t") {
				return fmt.Errorf("invalid shell environment record %d: UNSET requires exactly 2 tab-separated fields", i+1)
			}
		default:
			return fmt.Errorf("invalid shell environment record %d: expected SET or UNSET", i+1)
		}
		if key == "" {
			return fmt.Errorf("invalid shell environment record %d: empty key", i+1)
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
