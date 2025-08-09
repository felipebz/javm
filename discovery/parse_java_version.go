package discovery

import (
	"regexp"
	"strings"
)

var (
	reVersion = regexp.MustCompile(`version "([^"]+)"`)
	reVendor  = regexp.MustCompile(`(?m)^(OpenJDK|Java\(TM\))\b.*build`)
)

func ParseJavaVersionOutput(out string) map[string]string {
	md := make(map[string]string)

	if m := reVersion.FindStringSubmatch(out); len(m) > 1 {
		md["version"] = m[1]
	}

	if m := reVendor.FindStringSubmatch(out); len(m) > 1 {
		if m[1] == "OpenJDK" {
			md["vendor"] = "OpenJDK"
		} else {
			md["vendor"] = "Oracle"
		}
	}

	if strings.Contains(out, "JRE") {
		md["implementation"] = "JRE"
	} else {
		md["implementation"] = "JDK"
	}

	if strings.Contains(out, "64-Bit") {
		if strings.Contains(out, "aarch64") || strings.Contains(out, "arm64") {
			md["architecture"] = "arm64"
		} else {
			md["architecture"] = "x64"
		}
	} else {
		if strings.Contains(out, "arm") {
			md["architecture"] = "arm"
		} else {
			md["architecture"] = "x86"
		}
	}

	return md
}
