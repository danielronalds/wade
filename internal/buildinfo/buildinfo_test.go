package buildinfo

import (
	"runtime/debug"
	"testing"
)

func TestVersionWithReleaseVersion(t *testing.T) {
	originalVersion := releaseVersion
	t.Cleanup(func() { releaseVersion = originalVersion })

	for _, version := range []string{"v0.1.0-beta.1", "v0.1.0", "v0.0.0-snapshot.abcdef0"} {
		t.Run(version, func(t *testing.T) {
			releaseVersion = version
			if got := Version(); got != version {
				t.Fatalf("Version() = %q, want %q", got, version)
			}
		})
	}
}

func TestVersionWithoutReleaseVersion(t *testing.T) {
	originalVersion := releaseVersion
	t.Cleanup(func() { releaseVersion = originalVersion })
	releaseVersion = ""

	want := "(devel)"
	if buildInfo, ok := debug.ReadBuildInfo(); ok && buildInfo.Main.Version != "" {
		want = buildInfo.Main.Version
	}
	if got := Version(); got != want {
		t.Fatalf("Version() = %q, want %q", got, want)
	}
}
