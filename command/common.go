package command

import (
	"cmp"
	"context"
	"errors"
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
	// knownEmpty is set when the distribution exists but has no packages here.
	knownEmpty bool
}

type invalidPackageVersion struct {
	packageValue discoapi.Package
	err          error
}

func makePackageIndex(ctx context.Context, client PackagesClient, osFlag, archFlag, distributionFlag string) (*packageIndex, error) {
	pkgs, err := client.GetPackagesContext(ctx, osFlag, archFlag, distributionFlag, "")
	if err != nil {
		// The API rejects an unknown distribution with a 4xx; the distribution
		// list, not the API's message, decides whether that was the cause.
		if httpErr, ok := errors.AsType[*discoapi.HTTPError](err); ok && httpErr.StatusCode >= 400 && httpErr.StatusCode < 500 && distributionFlag != "" {
			if _, checkErr := checkDistribution(ctx, client, distributionFlag); checkErr != nil {
				return nil, checkErr
			}
		}
		return nil, NetworkError(err)
	}
	index := packageIndexFromPackages(pkgs)
	for _, invalid := range index.invalid {
		loggerFromContext(ctx).Debugf("ignoring package %q with invalid Java version %q: %v", invalid.packageValue.Id, invalid.packageValue.JavaVersion, invalid.err)
	}
	if len(pkgs) == 0 && distributionFlag != "" {
		// An empty result is the only success that needs the distribution
		// list, so the normal path makes no extra request.
		known, err := checkDistribution(ctx, client, distributionFlag)
		if err != nil {
			return nil, err
		}
		index.knownEmpty = known
	} else if distributionFlag != "" && !slices.ContainsFunc(pkgs, func(p discoapi.Package) bool { return p.Distribution == distributionFlag }) {
		// The API matches the name in any case, but installed identifiers keep
		// it as typed, so a name the list lacks exactly is rejected.
		if _, err := checkDistribution(ctx, client, distributionFlag); err != nil {
			return nil, err
		}
	}
	return index, nil
}

// checkDistribution returns an unknown distribution error when the Disco API's
// distribution list lacks name, and known when the list has it. A client
// without the list, or a list that cannot be fetched, proves neither.
func checkDistribution(ctx context.Context, client PackagesClient, name string) (known bool, err error) {
	lister, ok := client.(DistributionsClient)
	if !ok {
		return false, nil
	}
	distributions, err := lister.GetDistributionsContext(ctx)
	if err != nil {
		loggerFromContext(ctx).Debugf("could not check distribution %q: %v", name, err)
		return false, nil
	}
	if err := unknownDistributionError(name, distributions); err != nil {
		return false, err
	}
	return true, nil
}

// errUnknownDistribution marks a distribution name the Disco API does not offer.
var errUnknownDistribution = errors.New("unknown distribution")

// unknownDistributionError returns a usage error when name is not a Disco API
// distribution, suggesting the right case when only the case differs.
func unknownDistributionError(name string, distributions []discoapi.Distribution) error {
	for _, distribution := range distributions {
		if distribution.APIParameter == name {
			return nil
		}
	}
	hint := ""
	for _, distribution := range distributions {
		if strings.EqualFold(distribution.APIParameter, name) {
			hint = fmt.Sprintf("; did you mean %q?", distribution.APIParameter)
			break
		}
	}
	return UsageError(fmt.Errorf("%w %q; run 'javm ls-distributions' to see the available ones%s", errUnknownDistribution, name, hint))
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
		data, err := encodeShellEnvironment(out)
		if err != nil {
			return err
		}
		if err := os.WriteFile(fd3, data, 0600); err != nil {
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
	data, err := encodeShellEnvironment(out)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}
	if _, err := w.Write(data); err != nil {
		return shellIntegrationUnavailable()
	}
	return nil
}

// encodeShellEnvironment validates the records and terminates each one with a
// newline: `read` in bash and zsh drops a final line without one.
func encodeShellEnvironment(out []string) ([]byte, error) {
	if err := validateShellEnvironmentRecords(out); err != nil {
		return nil, err
	}
	var data []byte
	for _, line := range out {
		data = append(data, line...)
		data = append(data, '\n')
	}
	return data, nil
}

func validateShellEnvironmentRecords(out []string) error {
	for i, line := range out {
		if err := validateShellEnvironmentRecord(line); err != nil {
			return fmt.Errorf("invalid shell environment record %d: %w", i+1, err)
		}
	}
	return nil
}

// validateShellEnvironmentRecord accepts SET\t<key>\t<value> and UNSET\t<key>.
func validateShellEnvironmentRecord(line string) error {
	if strings.ContainsAny(line, "\r\n\x00") {
		return errors.New("CR, LF and NUL are not allowed")
	}
	operation, fields, ok := strings.Cut(line, "\t")
	if !ok {
		return errors.New("missing tab-separated fields")
	}
	var key string
	switch operation {
	case "SET":
		var value string
		key, value, ok = strings.Cut(fields, "\t")
		if !ok || strings.Contains(value, "\t") {
			return errors.New("SET requires exactly 3 tab-separated fields")
		}
	case "UNSET":
		key = fields
		if strings.Contains(key, "\t") {
			return errors.New("UNSET requires exactly 2 tab-separated fields")
		}
	default:
		return errors.New("expected SET or UNSET")
	}
	if key == "" {
		return errors.New("empty key")
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
