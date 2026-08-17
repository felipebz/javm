package state

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestAtomicWriteFileConcurrentWritersKeepCompleteData(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	const writers = 16
	var wg sync.WaitGroup
	errs := make(chan error, writers)
	wantValues := make([][]byte, writers)
	for i := 0; i < writers; i++ {
		value := []byte("{\"writer\":\"" + string(rune('a'+i)) + "\"}\n")
		wantValues[i] = value
		wg.Add(1)
		go func(data []byte) {
			defer wg.Done()
			if err := AtomicWriteFile(path, data, 0o600); err != nil {
				errs <- err
			}
		}(value)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("AtomicWriteFile() returned error: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	complete := false
	for _, want := range wantValues {
		if bytes.Equal(got, want) {
			complete = true
			break
		}
	}
	if !complete {
		t.Fatalf("concurrent write left incomplete data: %q", got)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".state-") {
			t.Fatalf("temporary state file was not removed: %s", entry.Name())
		}
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != 0o600 {
			t.Fatalf("state permissions = %o, want 600", got)
		}
	}
}

func TestWithFileLockSerializesCallbacks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	const writers = 8
	var wg sync.WaitGroup
	var active atomic.Int32
	var maximum atomic.Int32
	errs := make(chan error, writers)

	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := WithFileLock(path, func() error {
				current := active.Add(1)
				for {
					previous := maximum.Load()
					if current <= previous || maximum.CompareAndSwap(previous, current) {
						break
					}
				}
				time.Sleep(time.Millisecond)
				active.Add(-1)
				return nil
			})
			if err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("WithFileLock() returned error: %v", err)
	}
	if got := maximum.Load(); got != 1 {
		t.Fatalf("maximum callbacks in parallel = %d, want 1", got)
	}
}
