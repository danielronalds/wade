package server

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
)

func TestControllerStartsBackgroundProcess(t *testing.T) {
	temporaryDirectory := t.TempDir()
	homeDirectory := filepath.Join(temporaryDirectory, "home")
	stateDirectory := filepath.Join(temporaryDirectory, "state")
	pidPath := filepath.Join(temporaryDirectory, "server.pid")
	if err := os.MkdirAll(homeDirectory, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v, want nil", err)
	}

	executable := writeServerExecutable(t, temporaryDirectory, `#!/bin/sh
printf '%s' "$$" > "$WADE_TEST_PID_PATH"
printf '{"address":"test.localhost:1234","pid":%s}\n' "$$" >&3
exec sleep 30
`)
	t.Setenv("HOME", homeDirectory)
	t.Setenv("XDG_STATE_HOME", stateDirectory)
	t.Setenv("WADE_TEST_PID_PATH", pidPath)
	t.Setenv(serverReadyFileEnv, "99")

	var output bytes.Buffer
	controller := Controller{
		stdout: &output,
		executablePath: func() (string, error) {
			return executable, nil
		},
	}

	if err := controller.HandleArgs([]string{"server"}); err != nil {
		t.Fatalf("HandleArgs() error = %v, want nil", err)
	}

	pidContents, err := os.ReadFile(pidPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v, want nil", pidPath, err)
	}
	pid, err := strconv.Atoi(string(pidContents))
	if err != nil {
		t.Fatalf("Atoi(%q) error = %v, want nil", pidContents, err)
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		t.Fatalf("FindProcess(%d) error = %v, want nil", pid, err)
	}
	t.Cleanup(func() {
		_ = process.Signal(syscall.SIGTERM)
	})
	if err := process.Signal(syscall.Signal(0)); err != nil {
		t.Fatalf("background process is not running: %v", err)
	}

	logPath := filepath.Join(stateDirectory, "wade", "server.log")
	wantOutput := fmt.Sprintf(
		"WADE server listening on test.localhost:1234\nPID: %d\nLog: %s\n",
		pid,
		logPath,
	)
	if output.String() != wantOutput {
		t.Fatalf("output = %q, want %q", output.String(), wantOutput)
	}
	if _, err := os.Stat(logPath); err != nil {
		t.Fatalf("Stat(%q) error = %v, want nil", logPath, err)
	}
}

func TestControllerReportsBackgroundStartupFailure(t *testing.T) {
	temporaryDirectory := t.TempDir()
	homeDirectory := filepath.Join(temporaryDirectory, "home")
	stateDirectory := filepath.Join(temporaryDirectory, "state")
	if err := os.MkdirAll(homeDirectory, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v, want nil", err)
	}

	executable := writeServerExecutable(t, temporaryDirectory, `#!/bin/sh
printf '{"error":"address is already in use"}\n' >&3
exit 1
`)
	t.Setenv("HOME", homeDirectory)
	t.Setenv("XDG_STATE_HOME", stateDirectory)

	controller := Controller{
		stdout: &bytes.Buffer{},
		executablePath: func() (string, error) {
			return executable, nil
		},
	}

	err := controller.HandleArgs([]string{"server"})
	if err == nil || !strings.Contains(err.Error(), "failed to start WADE server: address is already in use") {
		t.Fatalf("HandleArgs() error = %v, want startup failure", err)
	}
}

func TestControllerRejectsUnexpectedArguments(t *testing.T) {
	controller := Controller{
		stdout: &bytes.Buffer{},
		executablePath: func() (string, error) {
			t.Fatal("executablePath() called for invalid arguments")
			return "", nil
		},
	}

	err := controller.HandleArgs([]string{"server", "unexpected"})
	if err == nil || err.Error() != "usage: wade server [--foreground]" {
		t.Fatalf("HandleArgs() error = %v, want usage error", err)
	}
}

func writeServerExecutable(t *testing.T, directory string, contents string) string {
	t.Helper()

	path := filepath.Join(directory, "wade-test-server")
	if err := os.WriteFile(path, []byte(contents), 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v, want nil", err)
	}
	return path
}
