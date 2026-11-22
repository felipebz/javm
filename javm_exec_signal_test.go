//go:build !windows

package main

import (
	"bytes"
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/felipebz/javm/discoapi"
	log "github.com/sirupsen/logrus"
)

func TestRootExecMapsChildSignalTo128PlusN(t *testing.T) {
	probe := createRootExecFixture(t)
	t.Setenv("JAVM_TEST_KILL_SELF", "1")

	_, stderr, code := executeRoot(t, "exec", "--jdk", "21", filepath.Base(probe))
	if code != 128+int(syscall.SIGTERM) || stderr != "" {
		t.Fatalf("exit %d, stderr %q, want %d and nothing from javm", code, stderr, 128+int(syscall.SIGTERM))
	}
}

// Ctrl-C reaches javm and the child, which share the terminal's process
// group. javm must leave the child alone and report how the child exited.
func TestRootExecLetsTheChildHandleInterrupt(t *testing.T) {
	probe := createRootExecFixture(t)
	ready := filepath.Join(t.TempDir(), "ready")
	t.Setenv("JAVM_TEST_READY_FILE", ready)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	var stdout, stderr bytes.Buffer
	root := newRootCommand(application{out: &stdout, err: &stderr, logger: log.New(), client: discoapi.NewClient()})
	root.SetArgs([]string{"exec", "--jdk", "21", filepath.Base(probe)})
	done := make(chan error, 1)
	go func() { done <- root.ExecuteContext(ctx) }()

	pid := waitForChildPID(t, ready)
	if err := syscall.Kill(os.Getpid(), syscall.SIGINT); err != nil {
		t.Fatal(err)
	}
	<-ctx.Done()
	// Give a context-bound child time to be killed before it handles SIGINT.
	time.Sleep(200 * time.Millisecond)
	if err := syscall.Kill(pid, syscall.SIGINT); err != nil {
		t.Fatalf("child %d is gone after javm received SIGINT: %v", pid, err)
	}

	select {
	case err := <-done:
		if code := exitCode(err); code != 5 || !strings.Contains(stdout.String(), "child cleanup") || stderr.Len() != 0 {
			t.Fatalf("exit %d, stdout %q, stderr %q, want the child's cleanup and exit 5", code, stdout.String(), stderr.String())
		}
	case <-time.After(10 * time.Second):
		t.Fatal("exec did not return after the child handled SIGINT")
	}
}

func waitForChildPID(t *testing.T, ready string) int {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if data, err := os.ReadFile(ready); err == nil && len(data) > 0 {
			pid, err := strconv.Atoi(string(data))
			if err != nil {
				t.Fatal(err)
			}
			return pid
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("child did not start")
	return 0
}
