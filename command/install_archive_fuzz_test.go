package command

import (
	"path"
	"path/filepath"
	"strings"
	"testing"
)

// FuzzArchiveEntryPath checks the extraction invariant behind safeArchiveName and entryPath: any accepted
// archive entry name, with or without stripping the archive root, resolves to a path inside the
// extraction root.
func FuzzArchiveEntryPath(f *testing.F) {
	for _, name := range []string{
		"jdk/bin/java", "jdk-25/Contents/Home/bin/java", "./jdk/lib", "jdk\\bin\\java.exe", "jdk/", ".", "..",
		"../evil", "jdk/../../evil", "/etc/passwd", "C:\\evil", "c:evil", "jdk/a/../../..", "jdk\\..\\..\\evil",
		"\x00", "jdk//bin", "...", "..jdk/bin",
	} {
		f.Add(name, false)
		f.Add(name, true)
	}
	root := filepath.Join(f.TempDir(), "root")
	f.Fuzz(func(t *testing.T, name string, strip bool) {
		clean, err := safeArchiveName(name)
		if err == nil {
			assertContainedArchivePath(t, root, name, clean)
		}

		state := newExtractionState(root, strip, extractionLimits{})
		rel, skip, err := state.entryPath(name)
		if err != nil || skip {
			return
		}
		assertContainedArchivePath(t, root, name, rel)
	})
}

func assertContainedArchivePath(t *testing.T, root, name, rel string) {
	t.Helper()
	if rel == ".." || strings.HasPrefix(rel, "../") || path.IsAbs(rel) || hasWindowsVolume(rel) ||
		strings.ContainsAny(rel, "\\\x00") {
		t.Fatalf("archive entry %q accepted as unsafe relative path %q", name, rel)
	}
	if target := filepath.Join(root, filepath.FromSlash(rel)); !pathWithinRoot(root, target) {
		t.Fatalf("archive entry %q resolves outside the extraction root: %q", name, target)
	}
}
