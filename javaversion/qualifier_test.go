package javaversion

import "testing"

func TestUnsafeQualifiersRejected(t *testing.T) {
	for _, qualifier := range []string{".", "..", "../../escaped", "../../pwned+x", `..\..\escaped`, "/tmp/evil", "C:evil", "evil\nSET\tPATH\tx", "evil\r", "evil\t", "evil\x00", "evil\x7f", "evil space", "evil／path", "evil@other"} {
		raw := qualifier + "@21.0.1"
		if _, err := ParseVersion(raw); err == nil {
			t.Errorf("ParseVersion(%q) accepted unsafe qualifier", raw)
		}
		if _, err := ParseRange(raw); err == nil {
			t.Errorf("ParseRange(%q) accepted unsafe qualifier", raw)
		}
	}
}

func TestSafeQualifiersPreserveVersionAndBuild(t *testing.T) {
	for _, qualifier := range []string{"temurin", "zulu", "graalvm_ce", "liberica", "sapmachine", "kona", "vendor.name-1", "vendor+x", "*"} {
		raw := qualifier + "@21.0.1+12"
		v, err := ParseVersion(raw)
		if err != nil {
			t.Fatal(err)
		}
		if v.Qualifier() != qualifier || v.WithoutBuild() != qualifier+"@21.0.1" {
			t.Fatalf("qualifier/build changed for %q: %q", raw, v.WithoutBuild())
		}
		r, err := ParseRange(qualifier + "@21")
		if err != nil || !r.Contains(v) {
			t.Fatalf("selector no longer matches %q: %v", raw, err)
		}
	}
	if _, err := ParseVersion("@21"); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseRange("@21"); err == nil {
		t.Fatal("empty explicit selector qualifier accepted")
	}
}
