// Package javaversion parses and compares Java version strings.
//
// Java versions use a numeric sequence (FEATURE.INTERIM.UPDATE.PATCH and
// potentially more elements), rather than the three-component SemVer model.
// The build number is part of the version used by javm when choosing an
// artifact.
package javaversion

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Version is a parsed Java version, optionally qualified by a distribution.
type Version struct {
	qualifier string
	raw       string

	numeric       []uint64
	prerelease    []identifier
	prereleaseRaw string
	build         uint64
	hasBuild      bool
	optional      []identifier
	optionalRaw   string
}

type identifier struct {
	raw       string
	numeric   uint64
	isNumeric bool
}

// Feature is the first Java version component.
func (v *Version) Feature() uint64 { return v.numericValue(0) }

// Interim is the second Java version component, or zero when absent.
func (v *Version) Interim() uint64 { return v.numericValue(1) }

// Update is the third Java version component, or zero when absent.
func (v *Version) Update() uint64 { return v.numericValue(2) }

// JavaPatch is the fourth Java version component, or zero when absent.
func (v *Version) JavaPatch() uint64 { return v.numericValue(3) }

// NumericSequence returns a copy of the Java numeric version sequence.
func (v *Version) NumericSequence() []uint64 {
	return append([]uint64(nil), v.numeric...)
}

// Major, Minor, and Patch are retained for compatibility with javm's former
// SemVer-shaped API. Patch means the third Java component here; use Update
// and JavaPatch when the Java version terminology matters.
func (v *Version) Major() uint64 { return v.Feature() }
func (v *Version) Minor() uint64 { return v.Interim() }
func (v *Version) Patch() uint64 { return v.Update() }

// Prerelease returns the original pre-release component, if present.
func (v *Version) Prerelease() string { return v.prereleaseRaw }

// BuildNumber returns the build number and whether the version included one.
func (v *Version) BuildNumber() (uint64, bool) { return v.build, v.hasBuild }

// Optional returns the original optional version information, if present.
func (v *Version) Optional() string { return v.optionalRaw }

// Qualifier returns the distribution qualifier, if present.
func (v *Version) Qualifier() string { return v.qualifier }

// Raw returns the original representation supplied to ParseVersion.
func (v *Version) Raw() string { return v.raw }

// String returns the original representation supplied to ParseVersion.
func (v *Version) String() string { return v.raw }

// Canonical returns a stable representation with trailing numeric zeroes
// removed. It is useful for de-duplicating equivalent versions while keeping
// build, pre-release, optional, and distribution information intact.
func (v *Version) Canonical() string {
	numeric := append([]uint64(nil), v.numeric...)
	for len(numeric) > 1 && numeric[len(numeric)-1] == 0 {
		numeric = numeric[:len(numeric)-1]
	}
	parts := make([]string, len(numeric))
	for i, value := range numeric {
		parts[i] = strconv.FormatUint(value, 10)
	}
	result := strings.Join(parts, ".")
	if v.prereleaseRaw != "" {
		result += "-" + v.prereleaseRaw
	}
	if v.hasBuild && v.build != 0 {
		result += "+" + strconv.FormatUint(v.build, 10)
	}
	if v.optionalRaw != "" {
		result += "-" + v.optionalRaw
	}
	if v.qualifier != "" {
		result = v.qualifier + "@" + result
	}
	return result
}

// Compare compares two qualified Java versions. Qualifiers retain javm's
// historic ordering behavior; versions from the same qualifier use Java
// version ordering.
func (v *Version) Compare(other *Version) int {
	if v.qualifier != other.qualifier {
		if v.qualifier > other.qualifier {
			return -1
		}
		return 1
	}
	return compareUnqualified(v, other)
}

func compareUnqualified(left, right *Version) int {
	max := max(len(left.numeric), len(right.numeric))
	for i := range max {
		lv := left.numericValue(i)
		rv := right.numericValue(i)
		if lv < rv {
			return -1
		}
		if lv > rv {
			return 1
		}
	}

	if result := compareIdentifiers(left.prerelease, right.prerelease, true); result != 0 {
		return result
	}

	// A missing build is treated as build zero. This keeps a plain release
	// below a positive build while making +0 equivalent to an omitted build.
	if left.build < right.build {
		return -1
	}
	if left.build > right.build {
		return 1
	}

	return compareIdentifiers(left.optional, right.optional, false)
}

func compareIdentifiers(left, right []identifier, numericRules bool) int {
	if len(left) == 0 && len(right) == 0 {
		return 0
	}
	if len(left) == 0 {
		return 1
	}
	if len(right) == 0 {
		return -1
	}

	for i := 0; i < min(len(left), len(right)); i++ {
		l, r := left[i], right[i]
		if numericRules && l.isNumeric != r.isNumeric {
			if l.isNumeric {
				return -1
			}
			return 1
		}
		if l.isNumeric && r.isNumeric {
			if l.numeric < r.numeric {
				return -1
			}
			if l.numeric > r.numeric {
				return 1
			}
			continue
		}
		if l.raw < r.raw {
			return -1
		}
		if l.raw > r.raw {
			return 1
		}
	}
	if len(left) < len(right) {
		return -1
	}
	if len(left) > len(right) {
		return 1
	}
	return 0
}

// LessThan reports whether v precedes other.
func (v *Version) LessThan(other *Version) bool { return v.Compare(other) < 0 }

// GreaterThan reports whether v follows other.
func (v *Version) GreaterThan(other *Version) bool { return v.Compare(other) > 0 }

// Equals reports whether both versions have the same Java version semantics,
// including qualifier, pre-release, build, and optional information.
func (v *Version) Equals(other *Version) bool { return v.Compare(other) == 0 }

func (v *Version) numericValue(index int) uint64 {
	if index >= len(v.numeric) {
		return 0
	}
	return v.numeric[index]
}

// ParseVersion parses either a Java version or <distribution>@<Java version>.
func ParseVersion(raw string) (*Version, error) {
	versionText := raw
	qualifier := ""
	if before, after, ok := strings.Cut(raw, "@"); ok {
		qualifier = before
		versionText = after
	}

	parsed, err := parseUnqualified(versionText)
	if err != nil {
		return nil, fmt.Errorf("%s is not a valid version", versionText)
	}
	parsed.qualifier = qualifier
	parsed.raw = raw
	return parsed, nil
}

func parseUnqualified(raw string) (*Version, error) {
	if raw == "" || strings.ContainsAny(raw, "@ \t\r\n") {
		return nil, fmt.Errorf("empty or invalid version")
	}

	base := raw
	optionalRaw := ""
	buildRaw := ""
	hasBuildSeparator := false
	if plus := strings.IndexByte(base, '+'); plus >= 0 {
		hasBuildSeparator = true
		if strings.IndexByte(base[plus+1:], '+') >= 0 {
			return nil, fmt.Errorf("multiple build separators")
		}
		buildAndOptional := base[plus+1:]
		if dash := strings.IndexByte(buildAndOptional, '-'); dash >= 0 {
			optionalRaw = buildAndOptional[dash+1:]
			buildAndOptional = buildAndOptional[:dash]
		}
		buildRaw = buildAndOptional
		if buildRaw == "" {
			return nil, fmt.Errorf("missing build number")
		}
		base = base[:plus]
	}

	prereleaseRaw := ""
	if dash := strings.IndexByte(base, '-'); dash >= 0 {
		prereleaseRaw = base[dash+1:]
		if prereleaseRaw == "" {
			return nil, fmt.Errorf("missing pre-release identifier")
		}
		base = base[:dash]
	}
	if optionalRaw == "" && strings.Contains(raw, "+") && strings.HasSuffix(raw, "-") {
		return nil, fmt.Errorf("missing optional information")
	}

	numeric, err := parseNumericSequence(base)
	if err != nil {
		return nil, err
	}

	parsed := &Version{
		numeric:       numeric,
		prereleaseRaw: prereleaseRaw,
		optionalRaw:   optionalRaw,
	}
	if prereleaseRaw != "" {
		parsed.prerelease, err = parseIdentifiers(prereleaseRaw)
		if err != nil {
			return nil, err
		}
	}
	if hasBuildSeparator {
		parsed.build, err = parseUint(buildRaw)
		if err != nil {
			return nil, err
		}
		parsed.hasBuild = true
	}
	if optionalRaw != "" {
		parsed.optional, err = parseIdentifiers(optionalRaw)
		if err != nil {
			return nil, err
		}
	}
	return parsed, nil
}

func parseNumericSequence(raw string) ([]uint64, error) {
	if raw == "" {
		return nil, fmt.Errorf("missing numeric version")
	}
	parts := strings.Split(raw, ".")
	result := make([]uint64, len(parts))
	for i, part := range parts {
		if part == "" {
			return nil, fmt.Errorf("empty numeric component")
		}
		value, err := parseUint(part)
		if err != nil {
			return nil, err
		}
		result[i] = value
	}
	return result, nil
}

func parseIdentifiers(raw string) ([]identifier, error) {
	parts := strings.Split(raw, ".")
	result := make([]identifier, len(parts))
	for i, part := range parts {
		if part == "" {
			return nil, fmt.Errorf("empty identifier")
		}
		for _, char := range part {
			if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') ||
				(char >= '0' && char <= '9') || char == '-' {
				continue
			}
			return nil, fmt.Errorf("invalid identifier")
		}
		result[i] = identifier{raw: part}
		if value, err := parseUint(part); err == nil {
			result[i].numeric = value
			result[i].isNumeric = true
		}
	}
	return result, nil
}

func parseUint(raw string) (uint64, error) {
	for _, char := range raw {
		if char < '0' || char > '9' {
			return 0, fmt.Errorf("not numeric")
		}
	}
	return strconv.ParseUint(raw, 10, 64)
}

// VersionSlice implements sort.Interface using Java version ordering.
type VersionSlice []*Version

func (s VersionSlice) Len() int      { return len(s) }
func (s VersionSlice) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s VersionSlice) Less(i, j int) bool {
	if comparison := s[i].Compare(s[j]); comparison != 0 {
		return comparison < 0
	}
	// Keep sorting deterministic for equivalent spellings such as 25 and
	// 25.0, without making those spellings semantically different.
	return s[i].raw < s[j].raw
}

// VersionPart identifies a Java numeric component used for display grouping.
type VersionPart int

const (
	VPFeature VersionPart = iota
	VPInterim
	VPUpdate
	VPJavaPatch

	// Compatibility names used by the existing CLI and API. VPPatch retains
	// its historical three-component display behavior; VPJavaPatch is the
	// fourth Java PATCH component.
	VPMajor = VPFeature
	VPMinor = VPInterim
	VPPatch = VPUpdate
)

// TrimTo returns the version through the requested numeric component.
func (v *Version) TrimTo(part VersionPart) string {
	count, ok := trimCount(part)
	if !ok {
		return v.raw
	}
	parts := make([]string, count)
	for i := range parts {
		parts[i] = strconv.FormatUint(v.numericValue(i), 10)
	}
	result := strings.Join(parts, ".")
	if v.qualifier != "" {
		result = v.qualifier + "@" + result
	}
	return result
}

func trimCount(part VersionPart) (int, bool) {
	switch part {
	case VPFeature:
		return 1, true
	case VPInterim:
		return 2, true
	case VPUpdate:
		return 3, true
	case VPJavaPatch:
		return 4, true
	default:
		return 0, false
	}
}

func (s VersionSlice) TrimTo(part VersionPart) VersionSlice {
	latest := make(map[string]*Version)
	for _, version := range s {
		key := versionTrimKey(version, part)
		previous, ok := latest[key]
		if !ok || version.GreaterThan(previous) ||
			(version.Compare(previous) == 0 && version.raw > previous.raw) {
			latest[key] = version
		}
	}

	result := make(VersionSlice, 0, len(latest))
	for _, version := range latest {
		result = append(result, version)
	}
	sort.Sort(result)
	return result
}

func versionTrimKey(version *Version, part VersionPart) string {
	count, ok := trimCount(part)
	if !ok {
		return version.Canonical()
	}
	parts := make([]string, count)
	for i := range parts {
		parts[i] = strconv.FormatUint(version.numericValue(i), 10)
	}
	return version.qualifier + ":" + strings.Join(parts, ".")
}
