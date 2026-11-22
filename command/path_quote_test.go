package command

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUnlinkShowsWindowsPathWithSingleBackslashes(t *testing.T) {
	linkTestHome(t)
	_, err := runLinkCommand(t, NewUnlinkCommand(), `C:\jdks\foo`)
	want := `no link matches "C:\jdks\foo"; see 'javm ls' for the available JDKs`
	if !errors.Is(err, ErrNotFound) || !strings.Contains(err.Error(), want) {
		t.Fatalf("unlink error = %v, want not found with %s", err, want)
	}
}

func TestUnlinkDropsControlCharactersFromPath(t *testing.T) {
	linkTestHome(t)
	_, err := runLinkCommand(t, NewUnlinkCommand(), "C:\\jdks\\\x1b[2Jfoo\a")
	if err == nil || !strings.Contains(err.Error(), `no link matches "C:\jdks\[2Jfoo"`) {
		t.Fatalf("unlink error = %v, want the path without control characters", err)
	}
}

func TestInstallDestinationShowsPathWithSingleBackslashes(t *testing.T) {
	dst := filepath.Join(t.TempDir(), `jdks\temurin`)
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatal(err)
	}
	err := installWithExtractor(context.Background(), "jdk.zip", dst, func(context.Context, string, string) error {
		t.Fatal("extractor ran for an existing destination")
		return nil
	}, false)
	want := `installation destination "` + dst + `" already exists; refusing to replace it`
	if err == nil || err.Error() != want {
		t.Fatalf("install error = %v, want %s", err, want)
	}
}
