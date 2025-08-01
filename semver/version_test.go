package semver

import (
	"reflect"
	"sort"
	"testing"
)

func TestSort(t *testing.T) {
	actual := asVersionSlice(t,
		"0.2.0", "a@1.8.10", "b@1.8.2", "0.1.20", "a@1.8.2", "0.1.10", "0.1.2", "1.9.0-10.1", "1.9.0-9.60")
	sort.Sort(sort.Reverse(VersionSlice(actual)))
	expected := asVersionSlice(t,
		"1.9.0-10.1", "1.9.0-9.60", "0.2.0", "0.1.20", "0.1.10", "0.1.2", "a@1.8.10", "a@1.8.2", "b@1.8.2")
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("actual: %v != expected: %v", actual, expected)
	}
}

func TestEquals(t *testing.T) {
	tests := []struct {
		name     string
		left     string
		right    string
		expected bool
	}{
		{
			name:     "same raw string",
			left:     "1.0.0",
			right:    "1.0.0",
			expected: true,
		},
		{
			name:     "different raw string",
			left:     "1.0.0",
			right:    "2.0.0",
			expected: false,
		},
		{
			name:     "same version with different qualifiers",
			left:     "a@1.0.0",
			right:    "b@1.0.0",
			expected: false,
		},
		{
			name:     "same qualifier with different versions",
			left:     "a@1.0.0",
			right:    "a@2.0.0",
			expected: false,
		},
		{
			name:     "identical with qualifier",
			left:     "a@1.0.0",
			right:    "a@1.0.0",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			left, err := ParseVersion(tt.left)
			if err != nil {
				t.Fatalf("failed to parse left version: %v", err)
			}
			right, err := ParseVersion(tt.right)
			if err != nil {
				t.Fatalf("failed to parse right version: %v", err)
			}

			actual := left.Equals(right)
			if actual != tt.expected {
				t.Errorf("Equals(%s, %s) = %v, expected %v", tt.left, tt.right, actual, tt.expected)
			}
		})
	}
}

func TestString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "1.0.0",
			expected: "1.0.0",
		},
		{
			input:    "a@1.0.0",
			expected: "a@1.0.0",
		},
		{
			input:    "1.2.3-beta.1",
			expected: "1.2.3-beta.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			v, err := ParseVersion(tt.input)
			if err != nil {
				t.Fatalf("failed to parse version: %v", err)
			}

			actual := v.String()
			if actual != tt.expected {
				t.Errorf("String() = %v, expected %v", actual, tt.expected)
			}
		})
	}
}

func TestTrimTo(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		part     VersionPart
		expected string
	}{
		{
			name:     "trim to major without qualifier",
			version:  "1.2.3",
			part:     VPMajor,
			expected: "1",
		},
		{
			name:     "trim to minor without qualifier",
			version:  "1.2.3",
			part:     VPMinor,
			expected: "1.2",
		},
		{
			name:     "trim to patch without qualifier",
			version:  "1.2.3",
			part:     VPPatch,
			expected: "1.2.3",
		},
		{
			name:     "trim to major with qualifier",
			version:  "a@1.2.3",
			part:     VPMajor,
			expected: "a@1",
		},
		{
			name:     "trim to minor with qualifier",
			version:  "a@1.2.3",
			part:     VPMinor,
			expected: "a@1.2",
		},
		{
			name:     "trim to patch with qualifier",
			version:  "a@1.2.3",
			part:     VPPatch,
			expected: "a@1.2.3",
		},
		{
			name:     "invalid part",
			version:  "1.2.3",
			part:     VersionPart(999),
			expected: "1.2.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := ParseVersion(tt.version)
			if err != nil {
				t.Fatalf("failed to parse version: %v", err)
			}

			actual := v.TrimTo(tt.part)
			if actual != tt.expected {
				t.Errorf("TrimTo(%v) = %v, expected %v", tt.part, actual, tt.expected)
			}
		})
	}
}

func TestAccessors(t *testing.T) {
	tests := []struct {
		version    string
		major      uint64
		minor      uint64
		patch      uint64
		prerelease string
	}{
		{
			version:    "1.2.3",
			major:      1,
			minor:      2,
			patch:      3,
			prerelease: "",
		},
		{
			version:    "0.0.1",
			major:      0,
			minor:      0,
			patch:      1,
			prerelease: "",
		},
		{
			version:    "10.20.30-beta.1",
			major:      10,
			minor:      20,
			patch:      30,
			prerelease: "beta.1",
		},
		{
			version:    "a@1.2.3",
			major:      1,
			minor:      2,
			patch:      3,
			prerelease: "",
		},
		{
			version:    "a@1.2.3-alpha.1",
			major:      1,
			minor:      2,
			patch:      3,
			prerelease: "alpha.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			v, err := ParseVersion(tt.version)
			if err != nil {
				t.Fatalf("failed to parse version: %v", err)
			}

			if v.Major() != tt.major {
				t.Errorf("Major() = %v, expected %v", v.Major(), tt.major)
			}
			if v.Minor() != tt.minor {
				t.Errorf("Minor() = %v, expected %v", v.Minor(), tt.minor)
			}
			if v.Patch() != tt.patch {
				t.Errorf("Patch() = %v, expected %v", v.Patch(), tt.patch)
			}
			if v.Prerelease() != tt.prerelease {
				t.Errorf("Prerelease() = %v, expected %v", v.Prerelease(), tt.prerelease)
			}
		})
	}
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectError   bool
		expectedError string
		qualifier     string
	}{
		{
			name:        "valid version without qualifier",
			input:       "1.2.3",
			expectError: false,
			qualifier:   "",
		},
		{
			name:        "valid version with qualifier",
			input:       "a@1.2.3",
			expectError: false,
			qualifier:   "a",
		},
		{
			name:          "invalid version",
			input:         "not-a-version",
			expectError:   true,
			expectedError: "not-a-version is not a valid version",
		},
		{
			name:          "invalid version with qualifier",
			input:         "a@not-a-version",
			expectError:   true,
			expectedError: "not-a-version is not a valid version",
		},
		{
			name:        "version with prerelease",
			input:       "1.2.3-beta.1",
			expectError: false,
			qualifier:   "",
		},
		{
			name:        "version with qualifier and prerelease",
			input:       "a@1.2.3-beta.1",
			expectError: false,
			qualifier:   "a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := ParseVersion(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				} else if err.Error() != tt.expectedError {
					t.Errorf("expected error %q, got %q", tt.expectedError, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if v.qualifier != tt.qualifier {
				t.Errorf("qualifier = %q, expected %q", v.qualifier, tt.qualifier)
			}

			if v.raw != tt.input {
				t.Errorf("raw = %q, expected %q", v.raw, tt.input)
			}
		})
	}
}

func TestVersionSliceTrimTo(t *testing.T) {
	tests := []struct {
		name     string
		versions []string
		part     VersionPart
		expected []string
	}{
		{
			name:     "trim to major",
			versions: []string{"1.2.3", "1.3.0", "2.0.0", "2.1.0"},
			part:     VPMajor,
			expected: []string{"1.3.0", "2.1.0"},
		},
		{
			name:     "trim to minor",
			versions: []string{"1.2.3", "1.2.4", "1.3.0", "2.1.0", "2.1.1"},
			part:     VPMinor,
			expected: []string{"1.2.4", "1.3.0", "2.1.1"},
		},
		{
			name:     "trim to patch",
			versions: []string{"1.2.3", "1.2.4", "1.3.0", "2.1.0"},
			part:     VPPatch,
			expected: []string{"1.2.3", "1.2.4", "1.3.0", "2.1.0"},
		},
		{
			name:     "with qualifiers",
			versions: []string{"a@1.2.3", "a@1.2.4", "b@1.2.3", "b@1.3.0"},
			part:     VPMajor,
			expected: []string{"b@1.3.0", "a@1.2.4"},
		},
		{
			name:     "mixed with and without qualifiers",
			versions: []string{"1.2.3", "a@1.2.4", "2.0.0", "a@2.1.0"},
			part:     VPMajor,
			expected: []string{"a@1.2.4", "a@2.1.0", "1.2.3", "2.0.0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			versions := asVersionSlice(t, tt.versions...)

			trimmed := versions.TrimTo(tt.part)

			var actual []string
			for _, v := range trimmed {
				actual = append(actual, v.raw)
			}

			if len(actual) != len(tt.expected) {
				t.Errorf("TrimTo returned %d versions, expected %d", len(actual), len(tt.expected))
				t.Errorf("actual: %v, expected: %v", actual, tt.expected)
				return
			}

			for i, v := range actual {
				if v != tt.expected[i] {
					t.Errorf("at index %d, got %q, expected %q", i, v, tt.expected[i])
				}
			}
		})
	}
}

func asVersionSlice(t *testing.T, slice ...string) (r VersionSlice) {
	for _, value := range slice {
		ver, err := ParseVersion(value)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		r = append(r, ver)
	}
	return
}
