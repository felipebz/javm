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

func (l *Version) LessThan(r *Version) bool {
	if l.qualifier == r.qualifier {
		return l.ver.LessThan(r.ver)
	}
	return l.qualifier > r.qualifier
}

func (l *Version) Equals(r *Version) bool {
	return l.raw == r.raw
}

func (t *Version) String() string {
	return t.raw
}

func (t *Version) TrimTo(part VersionPart) string {
	prefix := t.qualifier
	if prefix != "" {
		prefix += "@"
	}
	switch part {
	case VPMajor:
		return fmt.Sprintf("%v%v", prefix, t.ver.Major())
	case VPMinor:
		return fmt.Sprintf("%v%v.%v", prefix, t.ver.Major(), t.ver.Minor())
	case VPPatch:
		return fmt.Sprintf("%v%v.%v.%v", prefix, t.ver.Major(), t.ver.Minor(), t.ver.Patch())
	}
	return t.raw
}

func (t *Version) Major() uint64 {
	return t.ver.Major()
}

func (t *Version) Minor() uint64 {
	return t.ver.Minor()
}

func (t *Version) Patch() uint64 {
	return t.ver.Patch()
}

func (t *Version) Prerelease() string {
	return t.ver.Prerelease()
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

func (c VersionSlice) Len() int {
	return len(c)
}
func (c VersionSlice) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}
func (c VersionSlice) Less(i, j int) bool {
	return c[i].LessThan(c[j])
}

type VersionPart int

const (
	VPMajor VersionPart = iota
	VPMinor
	VPPatch
)

func (c VersionSlice) TrimTo(part VersionPart) VersionSlice {
	latest := make(map[string]*Version)
	for _, v := range c {
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
