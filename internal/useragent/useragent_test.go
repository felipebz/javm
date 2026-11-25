package useragent

import "testing"

func TestFormat(t *testing.T) {
	for version, want := range map[string]string{
		"1.0.0": "javm/1.0.0 (+https://github.com/felipebz/javm)",
		"1.0.1-0.20261125120000-abcdef123456+dirty": "javm/1.0.1-0.20261125120000-abcdef123456+dirty (+https://github.com/felipebz/javm)",
		"":            "javm/dev (+https://github.com/felipebz/javm)",
		"1.0 beta":    "javm/dev (+https://github.com/felipebz/javm)",
		"1.0/2":       "javm/dev (+https://github.com/felipebz/javm)",
		"1.0(x)":      "javm/dev (+https://github.com/felipebz/javm)",
		"1.0\r\nX: y": "javm/dev (+https://github.com/felipebz/javm)",
		"1.0\x1b":     "javm/dev (+https://github.com/felipebz/javm)",
	} {
		if got := Format(version); got != want {
			t.Errorf("Format(%q) = %q, want %q", version, got, want)
		}
	}
}

func TestOrDefault(t *testing.T) {
	if got := OrDefault(""); got != "javm/dev (+https://github.com/felipebz/javm)" {
		t.Errorf("OrDefault(\"\") = %q", got)
	}
	if got := OrDefault("javm/1.0.0 (+https://github.com/felipebz/javm)"); got != "javm/1.0.0 (+https://github.com/felipebz/javm)" {
		t.Errorf("OrDefault kept %q", got)
	}
}
