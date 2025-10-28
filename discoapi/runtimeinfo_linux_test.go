package discoapi

import (
	"errors"
	"os"
	"syscall"
	"testing"
	"time"
)

func withMocks(t *testing.T,
	mockFileExists func(string) bool,
	mockSafeLdd func() ([]byte, bool),
) func() {
	t.Helper()

	origFileExists := fileExistsFn
	origSafeLdd := safeLddVersionFn
	origStatFn := statFn

	// mock fileExistsFn
	if mockFileExists != nil {
		fileExistsFn = mockFileExists
	} else {
		fileExistsFn = origFileExists
	}

	// mock safeLddVersionFn
	if mockSafeLdd != nil {
		safeLddVersionFn = mockSafeLdd
	} else {
		safeLddVersionFn = origSafeLdd
	}

	statFn = func(string) (os.FileInfo, error) {
		return nil, errors.New("stat not allowed in test")
	}

	return func() {
		fileExistsFn = origFileExists
		safeLddVersionFn = origSafeLdd
		statFn = origStatFn
	}
}

type fakeFileInfo struct {
	mode os.FileMode
	dir  bool
	sys  *syscall.Stat_t
}

func (f fakeFileInfo) Name() string       { return "ldd" }
func (f fakeFileInfo) Size() int64        { return 0 }
func (f fakeFileInfo) Mode() os.FileMode  { return f.mode }
func (f fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (f fakeFileInfo) IsDir() bool        { return f.dir }
func (f fakeFileInfo) Sys() any           { return f.sys }

//
// Test cases
//

func TestIsMuslLibc_MuslFastPath(t *testing.T) {
	cleanup := withMocks(t,
		func(p string) bool {
			return p == "/lib/ld-musl-x86_64.so.1"
		},
		func() ([]byte, bool) {
			t.Fatal("ldd should not be called for musl fast-path")
			return nil, false
		},
	)
	defer cleanup()

	if !isMuslLibc() {
		t.Errorf("expected true (musl fast-path)")
	}
}

func TestIsMuslLibc_GlibcFastPath(t *testing.T) {
	cleanup := withMocks(t,
		func(p string) bool {
			return p == "/lib64/ld-linux-x86-64.so.2"
		},
		func() ([]byte, bool) {
			t.Fatal("ldd should not be called for glibc fast-path")
			return nil, false
		},
	)
	defer cleanup()

	if isMuslLibc() {
		t.Errorf("expected false (glibc fast-path)")
	}
}

func TestIsMuslLibc_LddFallback_Musl(t *testing.T) {
	cleanup := withMocks(t,
		func(p string) bool {
			return false
		},
		func() ([]byte, bool) {
			return []byte("musl libc (x86_64)\nVersion 1.2.5\n"), true
		},
	)
	defer cleanup()

	if !isMuslLibc() {
		t.Errorf("expected true (musl via ldd fallback)")
	}
}

func TestIsMuslLibc_LddFallback_Glibc(t *testing.T) {
	cleanup := withMocks(t,
		func(p string) bool {
			return false
		},
		func() ([]byte, bool) {
			return []byte("GNU C Library (GNU libc) version 2.39\nFree Software Foundation..."), true
		},
	)
	defer cleanup()

	if isMuslLibc() {
		t.Errorf("expected false (glibc via ldd fallback)")
	}
}

func TestIsMuslLibc_Inconclusive(t *testing.T) {
	cleanup := withMocks(t,
		func(p string) bool {
			return false
		},
		func() ([]byte, bool) {
			return nil, false
		},
	)
	defer cleanup()

	if isMuslLibc() {
		t.Errorf("expected false (inconclusive defaults to false)")
	}
}
