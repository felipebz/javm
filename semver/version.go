package semver

import (
	"fmt"
	"github.com/Masterminds/semver/v3"
	"sort"
	"strings"
)

type Version struct {
	qualifier string
	raw       string
	ver       *semver.Version
}

func (v *Version) LessThan(other *Version) bool {
	if v.qualifier == other.qualifier {
		return v.ver.LessThan(other.ver)
	}
	return v.qualifier > other.qualifier
}

func (v *Version) Equals(other *Version) bool {
	return v.raw == other.raw
}

func (v *Version) String() string {
	return v.raw
}

func (v *Version) TrimTo(part VersionPart) string {
	prefix := v.qualifier
	if prefix != "" {
		prefix += "@"
	}
	switch part {
	case VPMajor:
		return fmt.Sprintf("%v%v", prefix, v.ver.Major())
	case VPMinor:
		return fmt.Sprintf("%v%v.%v", prefix, v.ver.Major(), v.ver.Minor())
	case VPPatch:
		return fmt.Sprintf("%v%v.%v.%v", prefix, v.ver.Major(), v.ver.Minor(), v.ver.Patch())
	}
	return v.raw
}

func (v *Version) Major() uint64 {
	return v.ver.Major()
}

func (v *Version) Minor() uint64 {
	return v.ver.Minor()
}

func (v *Version) Patch() uint64 {
	return v.ver.Patch()
}

func (v *Version) Prerelease() string {
	return v.ver.Prerelease()
}

func ParseVersion(raw string) (*Version, error) {
	p := new(Version)
	p.raw = raw
	// selector can be either <version> or <qualifier>@<version>
	if strings.Contains(raw, "@") {
		p.qualifier = raw[0:strings.Index(raw, "@")]
		raw = raw[strings.Index(raw, "@")+1:]
	}
	parsed, err := semver.NewVersion(raw)
	if err != nil {
		return nil, fmt.Errorf("%s is not a valid version", raw)
	}
	p.ver = parsed
	return p, nil
}

type VersionSlice []*Version

// impl sort.Interface:

func (s VersionSlice) Len() int {
	return len(s)
}
func (s VersionSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s VersionSlice) Less(i, j int) bool {
	return s[i].LessThan(s[j])
}

type VersionPart int

const (
	VPMajor VersionPart = iota
	VPMinor
	VPPatch
)

func (s VersionSlice) TrimTo(part VersionPart) VersionSlice {
	latest := make(map[string]*Version)
	for _, v := range s {
		key := versionTrimKey(v, part)
		if prev, ok := latest[key]; !ok || v.ver.GreaterThan(prev.ver) {
			latest[key] = v
		}
	}

	result := make(VersionSlice, 0, len(latest))
	for _, v := range latest {
		result = append(result, v)
	}
	sort.Sort(result)
	return result
}

func versionTrimKey(v *Version, part VersionPart) string {
	switch part {
	case VPMajor:
		return fmt.Sprintf("%s:%d", v.qualifier, v.Major())
	case VPMinor:
		return fmt.Sprintf("%s:%d.%d", v.qualifier, v.Major(), v.Minor())
	case VPPatch:
		return fmt.Sprintf("%s:%d.%d.%d", v.qualifier, v.Major(), v.Minor(), v.Patch())
	default:
		return v.String()
	}
}
