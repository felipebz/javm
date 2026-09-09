package javaversion

import (
	"slices"
	"testing"
)

func TestJavaVersionOrdering(t *testing.T) {
	tests := []struct {
		name  string
		left  string
		right string
		want  int
	}{
		{name: "feature before update", left: "25", right: "25.0.1", want: -1},
		{name: "update before patch", left: "25.0.4", right: "25.0.4.1", want: -1},
		{name: "patch before next update", left: "25.0.4.1", right: "25.0.5", want: -1},
		{name: "build numbers", left: "25.0.4+6", right: "25.0.4+7", want: -1},
		{name: "numeric sequence before build", left: "25.0.4+7", right: "25.0.4.1+1", want: -1},
		{name: "patch build numbers", left: "25.0.4.1+1", right: "25.0.4.1+2", want: -1},
		{name: "additional numeric element", left: "25.0.4.1", right: "25.0.4.1.2", want: -1},
		{name: "build before optional information", left: "25+1", right: "25+2-linux", want: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			left, err := ParseVersion(tt.left)
			if err != nil {
				t.Fatal(err)
			}
			right, err := ParseVersion(tt.right)
			if err != nil {
				t.Fatal(err)
			}
			got := left.Compare(right)
			if got != tt.want {
				t.Fatalf("Compare(%q, %q) = %d, want %d", tt.left, tt.right, got, tt.want)
			}
		})
	}
}

func TestJavaVersionTrailingZeroesAreEquivalent(t *testing.T) {
	left, err := ParseVersion("25.0.4.1")
	if err != nil {
		t.Fatal(err)
	}
	right, err := ParseVersion("25.0.4.1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if !left.Equals(right) {
		t.Fatalf("trailing zeroes changed equality: %q != %q", left, right)
	}
	if left.Canonical() != right.Canonical() {
		t.Fatalf("canonical values differ: %q != %q", left.Canonical(), right.Canonical())
	}
}

func TestJavaVersionTrimToJavaPatch(t *testing.T) {
	version, err := ParseVersion("temurin@25.0.4.1+1")
	if err != nil {
		t.Fatal(err)
	}
	if got := version.TrimTo(VPJavaPatch); got != "temurin@25.0.4.1" {
		t.Fatalf("TrimTo(VPJavaPatch) = %q", got)
	}
}

func TestJavaVersionWithoutBuild(t *testing.T) {
	version, err := ParseVersion("temurin@25.0.4.1+1")
	if err != nil {
		t.Fatal(err)
	}
	if got := version.WithoutBuild(); got != "temurin@25.0.4.1" {
		t.Fatalf("WithoutBuild() = %q", got)
	}
}

func TestJavaVersionSameReleaseIgnoresBuild(t *testing.T) {
	withoutBuild, err := ParseVersion("temurin@25.0.4.1")
	if err != nil {
		t.Fatal(err)
	}
	withBuild, err := ParseVersion("temurin@25.0.4.1+1")
	if err != nil {
		t.Fatal(err)
	}
	if !withoutBuild.SameRelease(withBuild) {
		t.Fatalf("versions should identify the same release: %q and %q", withoutBuild, withBuild)
	}
	if withoutBuild.Equals(withBuild) {
		t.Fatal("versions with different build information must not be fully equal")
	}
}

func TestJavaVersionPreservesCompleteRepresentation(t *testing.T) {
	version, err := ParseVersion("temurin@25.0.4.1.2-ea+7-linux")
	if err != nil {
		t.Fatal(err)
	}
	if version.Qualifier() != "temurin" {
		t.Fatalf("qualifier = %q", version.Qualifier())
	}
	if !slices.Equal(version.NumericSequence(), []uint64{25, 0, 4, 1, 2}) {
		t.Fatalf("numeric sequence = %v", version.NumericSequence())
	}
	if version.Prerelease() != "ea" {
		t.Fatalf("pre-release = %q", version.Prerelease())
	}
	if build, ok := version.BuildNumber(); !ok || build != 7 {
		t.Fatalf("build = %d, present = %v", build, ok)
	}
	if version.Optional() != "linux" {
		t.Fatalf("optional information = %q", version.Optional())
	}
}
