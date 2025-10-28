package discoapi

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

var statFn = os.Stat

var fileExistsFn = func(p string) bool {
	st, err := statFn(p)
	return err == nil && !st.IsDir()
}

var safeLddVersionFn = safeLddVersion

func isMuslLibc() bool {
	if looksLikeMuslFilesystem() {
		return true
	}

	if looksLikeGlibcFilesystem() {
		return false
	}

	out, ok := safeLddVersionFn()
	if ok {
		if bytes.Contains(out, []byte("musl libc")) {
			return true
		}
	}

	return false
}

func looksLikeMuslFilesystem() bool {
	candidates := []string{
		"/lib/ld-musl-x86_64.so.1",
		"/lib/ld-musl-aarch64.so.1",
		"/usr/lib/ld-musl-x86_64.so.1",
		"/usr/lib/ld-musl-aarch64.so.1",
	}
	for _, p := range candidates {
		if fileExistsFn(p) {
			return true
		}
	}
	return false
}

func looksLikeGlibcFilesystem() bool {
	candidates := []string{
		"/lib64/ld-linux-x86-64.so.2",
		"/lib/ld-linux-x86-64.so.2",
		"/usr/lib/ld-linux-x86-64.so.2",
		"/lib/ld-linux-aarch64.so.1",
		"/lib64/ld-linux-aarch64.so.1",
		"/usr/lib/ld-linux-aarch64.so.1",
	}
	for _, p := range candidates {
		if fileExistsFn(p) {
			return true
		}
	}
	return false
}

func safeLddVersion() ([]byte, bool) {
	allowed := []string{
		"/usr/bin/ldd",
		"/bin/ldd",
	}

	lddPath, err := exec.LookPath("ldd")
	if err != nil || lddPath == "" {
		for _, cand := range allowed {
			if st, err := statFn(cand); err == nil && !st.IsDir() {
				lddPath = cand
				break
			}
		}
		if lddPath == "" {
			return nil, false
		}
	}

	resolved, err := filepath.EvalSymlinks(lddPath)
	if err != nil {
		return nil, false
	}
	if !isInList(resolved, allowed) {
		return nil, false
	}

	fi, err := statFn(resolved)
	if err != nil {
		return nil, false
	}
	if fi.IsDir() {
		return nil, false
	}

	if fi.Mode().Perm()&0o022 != 0 {
		return nil, false
	}

	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return nil, false
	}
	if st.Uid != 0 {
		return nil, false
	}

	cmd := exec.Command(resolved, "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, false
	}
	return out, true
}

func isInList(s string, list []string) bool {
	for _, v := range list {
		if s == v {
			return true
		}
	}
	return false
}
