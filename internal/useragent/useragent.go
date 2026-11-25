// Package useragent builds the User-Agent header javm sends on every HTTP
// request. It identifies the program, its version and the project, and
// nothing that tells one user from another.
package useragent

import "strings"

const projectURL = "https://github.com/felipebz/javm"

// Format returns the header value for version. A version that is not a valid
// RFC 9110 token is replaced by "dev", since Go rejects invalid header values
// at request time.
func Format(version string) string {
	if !isToken(version) {
		version = "dev"
	}
	return "javm/" + version + " (+" + projectURL + ")"
}

// OrDefault returns value, or the header of a dev build when value is empty,
// so no request goes out with Go's default User-Agent.
func OrDefault(value string) string {
	if value == "" {
		return Format("")
	}
	return value
}

func isToken(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !('a' <= r && r <= 'z' || 'A' <= r && r <= 'Z' || '0' <= r && r <= '9' || strings.ContainsRune("!#$%&'*+-.^_`|~", r)) {
			return false
		}
	}
	return true
}
