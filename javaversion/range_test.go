package javaversion

import "testing"

func TestRangePreservesCommonSelectors(t *testing.T) {
	tests := []struct {
		selector string
		version  string
		want     bool
	}{
		{selector: "1.8", version: "1.8.72", want: true},
		{selector: "1.7", version: "1.8.0", want: false},
		{selector: "~1.8", version: "1.8.99", want: true},
		{selector: "~1.8", version: "1.9.0", want: false},
		{selector: "~17.0.10", version: "17.0.10+9", want: true},
		{selector: "~17.0.10", version: "17.1.0", want: false},
		{selector: "temurin@1.8", version: "temurin@1.8.72", want: true},
		{selector: "temurin@1.8", version: "zulu@1.8.72", want: false},
		{selector: "1.8", version: "temurin@1.8.72", want: true},
		{selector: ">=1.7 <=1.8.75", version: "1.8.72", want: true},
		{selector: ">=1.7 <=1.8.75", version: "1.8.80", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.selector+"/"+tt.version, func(t *testing.T) {
			rangeValue, err := ParseRange(tt.selector)
			if err != nil {
				t.Fatal(err)
			}
			version, err := ParseVersion(tt.version)
			if err != nil {
				t.Fatal(err)
			}
			if got := rangeValue.Contains(version); got != tt.want {
				t.Fatalf("Contains(%q) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}

func TestRangeMatchesJavaBuildsAndPatchComponents(t *testing.T) {
	tests := []struct {
		selector string
		version  string
		want     bool
	}{
		{selector: "25", version: "25.0.4.1+1", want: true},
		{selector: "25.0.4", version: "25.0.4+7", want: true},
		{selector: "25.0.4", version: "25.0.4.1+1", want: false},
		{selector: "25.0.4.1", version: "25.0.4.1+1", want: true},
		{selector: "25.0.4.1+1", version: "25.0.4.1+2", want: false},
		{selector: "25.0.4.1", version: "25.0.4.1.2", want: false},
	}

	for _, tt := range tests {
		rangeValue, err := ParseRange(tt.selector)
		if err != nil {
			t.Fatalf("ParseRange(%q): %v", tt.selector, err)
		}
		version, err := ParseVersion(tt.version)
		if err != nil {
			t.Fatalf("ParseVersion(%q): %v", tt.version, err)
		}
		if got := rangeValue.Contains(version); got != tt.want {
			t.Errorf("%q contains %q = %v, want %v", tt.selector, tt.version, got, tt.want)
		}
	}
}
