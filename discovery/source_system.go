package discovery

import (
	"os"
	"path/filepath"
	"runtime"
)

type SystemSource struct {
	locations []string // Optional override for tests
}

func NewSystemSource() *SystemSource {
	return &SystemSource{}
}

func (s *SystemSource) Name() string {
	return "system"
}

func (s *SystemSource) getLocations() []string {
	if len(s.locations) > 0 {
		return s.locations
	}

	switch runtime.GOOS {
	case "linux":
		return []string{
			"/usr/lib/jvm",
			"/opt/java",
		}
	case "darwin":
		return []string{
			"/Library/Java/JavaVirtualMachines",
		}
	case "windows":
		return []string{
			filepath.Join(os.Getenv("ProgramFiles"), "Java"),
			filepath.Join(os.Getenv("ProgramFiles(x86)"), "Java"),
		}
	}
	return nil
}

func (s *SystemSource) Discover() ([]JDK, error) {
	return ScanLocationsForJDKs(s.getLocations(), s.Name())
}
