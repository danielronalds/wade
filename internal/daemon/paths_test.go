package daemon

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePathsUsesDefaultStateDirectory(t *testing.T) {
	homeDirectory := t.TempDir()
	t.Setenv("HOME", homeDirectory)
	t.Setenv("XDG_STATE_HOME", "")

	paths, err := ResolvePaths()
	if err != nil {
		t.Fatalf("ResolvePaths() error = %v, want nil", err)
	}

	wantStateDirectory := filepath.Join(homeDirectory, ".local", "state", "wade")
	if paths.StateDirectory != wantStateDirectory {
		t.Fatalf("StateDirectory = %q, want %q", paths.StateDirectory, wantStateDirectory)
	}
	if paths.LogPath != filepath.Join(wantStateDirectory, "server.log") {
		t.Fatalf("LogPath = %q, want path beneath state directory", paths.LogPath)
	}
	if paths.SocketPath != filepath.Join(wantStateDirectory, "server.sock") {
		t.Fatalf("SocketPath = %q, want path beneath state directory", paths.SocketPath)
	}
	if _, err := os.Stat(paths.StateDirectory); !os.IsNotExist(err) {
		t.Fatalf("ResolvePaths() created state directory, Stat() error = %v", err)
	}
}

func TestResolvePathsUsesXDGStateHome(t *testing.T) {
	stateRoot := t.TempDir()
	t.Setenv("XDG_STATE_HOME", stateRoot)

	paths, err := ResolvePaths()
	if err != nil {
		t.Fatalf("ResolvePaths() error = %v, want nil", err)
	}

	want := filepath.Join(stateRoot, "wade")
	if paths.StateDirectory != want {
		t.Fatalf("StateDirectory = %q, want %q", paths.StateDirectory, want)
	}
}

func TestEnsureStateDirectoryAppliesRestrictivePermissions(t *testing.T) {
	stateDirectory := filepath.Join(t.TempDir(), "wade")
	if err := os.Mkdir(stateDirectory, 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v, want nil", err)
	}

	paths := Paths{StateDirectory: stateDirectory, LogPath: filepath.Join(stateDirectory, "server.log")}
	logFile, err := paths.openLog()
	if err != nil {
		t.Fatalf("openLog() error = %v, want nil", err)
	}
	_ = logFile.Close()

	stateInfo, err := os.Stat(stateDirectory)
	if err != nil {
		t.Fatalf("Stat(state directory) error = %v, want nil", err)
	}
	if stateInfo.Mode().Perm() != 0o700 {
		t.Fatalf("state directory mode = %o, want 700", stateInfo.Mode().Perm())
	}
	logInfo, err := os.Stat(paths.LogPath)
	if err != nil {
		t.Fatalf("Stat(log) error = %v, want nil", err)
	}
	if logInfo.Mode().Perm() != 0o600 {
		t.Fatalf("log mode = %o, want 600", logInfo.Mode().Perm())
	}
}
