package command

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/felipebz/javm/discoapi"
)

func TestRunInstallRejectsHostileDistribution(t *testing.T) {
	for _, distribution := range []string{"../../escaped", "../../pwned+x", `..\..\escaped`, "evil\nSET\tPATH\tx"} {
		t.Run(distribution, func(t *testing.T) {
			base := t.TempDir()
			home := filepath.Join(base, "home")
			t.Setenv("JAVM_HOME", home)
			client := &versionSelectionClient{packages: []discoapi.Package{{Id: "hostile", Distribution: distribution, JavaVersion: "21.0.1"}}}
			if _, err := runInstall(context.Background(), client, "21", ""); err == nil {
				t.Fatal("hostile package installed")
			}
			if client.infoID != "" {
				t.Fatal("hostile package reached artifact lookup")
			}
			for _, path := range []string{filepath.Join(base, "escaped@21.0.1"), filepath.Join(base, "pwned"), filepath.Join(home, "jdk")} {
				if _, err := os.Stat(path); !os.IsNotExist(err) {
					t.Fatalf("unexpected destination %q: %v", path, err)
				}
			}
		})
	}
}

func TestRunInstallBenignDistributionStaysManaged(t *testing.T) {
	t.Setenv("JAVM_HOME", t.TempDir())
	home := os.Getenv("JAVM_HOME")
	archive := makeZipArchive(t, []zipTestEntry{{name: javaArchivePath(), body: "java", mode: 0755}})
	client := &versionSelectionClient{packages: []discoapi.Package{{Id: "benign", Distribution: "temurin", JavaVersion: "21.0.1+12"}}, archive: archive}
	version, err := runInstall(context.Background(), client, "21", "")
	if err != nil {
		t.Fatal(err)
	}
	if version != "temurin@21.0.1+12" {
		t.Fatalf("version = %q", version)
	}
	if err := assertJavaDistribution(filepath.Join(home, "jdk", "temurin@21.0.1"), runtime.GOOS); err != nil {
		t.Fatal(err)
	}
}

func TestManagedInstallDestinationRejectsUnsafeComponents(t *testing.T) {
	for _, name := range []string{"", ".", "..", "../escape", `..\escape`, "C:escape", "evil\x00", "evil\n", "evil\t", "evil\x7f"} {
		if _, err := managedInstallDestination(t.TempDir(), name); err == nil {
			t.Errorf("accepted %q", name)
		}
	}
}
