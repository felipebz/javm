package discovery

import (
	"regexp"
	"strings"
)

var wantedKeys = map[string]string{
	"java.vendor":  "vendor",
	"java.version": "version",
	"os.arch":      "architecture",
}

var lineRE = regexp.MustCompile(`^\s*([a-zA-Z0-9_.]+)\s*=\s*(.*)\s*$`)

func ParseJavaVersionOutput(out string) map[string]string {
	props := parseProps(out)

	res := make(map[string]string)
	for k, v := range wantedKeys {
		r, ok := props[k]
		if ok && r != "" {
			res[v] = r
		}
	}

	return res
}

func parseProps(s string) map[string]string {
	m := make(map[string]string, 64)
	for line := range strings.SplitSeq(s, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" || strings.HasSuffix(line, ":") {
			continue
		}
		if g := lineRE.FindStringSubmatch(line); g != nil {
			key := strings.TrimSpace(g[1])
			val := strings.TrimSpace(g[2])
			m[key] = val
		}
	}
	return m
}
