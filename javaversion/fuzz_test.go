package javaversion

import "testing"

var versionSeeds = []string{
	"25", "25.0.4", "25.0.4.1", "25.0.4+7", "25.0.4.1+2", "25+2-linux", "25-ea", "25-ea.1", "25-beta",
	"8.0.514+1", "1.8.0_72", "temurin@25.0.5.1", "zulu@8.0.514+1", "system@21.0.1", "", "@", "25..1", "+", "-",
}

var rangeSeeds = []string{
	"25", "temurin@25", "zulu@~8.0.400", "~21.0.2", "^17.0.1", ">=21 <22", "17 - 21", "17 || 21", "*@>=17 <22",
	"21.0.x", "temurin@", "*", ">=", "||", "1 - ", "~", "^", "<=25.0.4+7",
}

// FuzzParseVersion checks that parsing never panics and that ordering is reflexive, antisymmetric, and
// consistent with Canonical, which is used to de-duplicate equivalent versions.
func FuzzParseVersion(f *testing.F) {
	for _, left := range versionSeeds {
		f.Add(left, "25.0.4")
	}
	f.Fuzz(func(t *testing.T, leftRaw, rightRaw string) {
		left, err := ParseVersion(leftRaw)
		if err != nil {
			return
		}
		if left.Compare(left) != 0 {
			t.Fatalf("%q does not compare equal to itself", leftRaw)
		}

		canonical, err := ParseVersion(left.Canonical())
		if err != nil {
			t.Fatalf("Canonical() of %q is not parseable: %q: %v", leftRaw, left.Canonical(), err)
		}
		if canonical.Canonical() != left.Canonical() {
			t.Fatalf("Canonical() of %q is not stable: %q then %q", leftRaw, left.Canonical(), canonical.Canonical())
		}

		right, err := ParseVersion(rightRaw)
		if err != nil {
			return
		}
		if sign(left.Compare(right)) != -sign(right.Compare(left)) {
			t.Fatalf("ordering of %q and %q is not antisymmetric", leftRaw, rightRaw)
		}
	})
}

// FuzzParseRange checks that parsing selectors and matching versions never panic, and that a parsed
// selector keeps its original text.
func FuzzParseRange(f *testing.F) {
	for _, selector := range rangeSeeds {
		f.Add(selector, "25.0.4")
	}
	f.Fuzz(func(t *testing.T, selector, versionRaw string) {
		parsed, err := ParseRange(selector)
		if err != nil {
			return
		}
		if parsed.String() != selector {
			t.Fatalf("String() = %q, want %q", parsed.String(), selector)
		}
		version, err := ParseVersion(versionRaw)
		if err != nil {
			return
		}
		parsed.Contains(version)
	})
}

func sign(value int) int {
	switch {
	case value < 0:
		return -1
	case value > 0:
		return 1
	default:
		return 0
	}
}
