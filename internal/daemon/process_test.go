package daemon

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

const testDaemonArgument = "test-daemon"

func TestManagerStartsDetachedProcessAndWaitsForReadiness(t *testing.T) {
	temporaryDirectory := shortStateRoot(t)
	homeDirectory := filepath.Join(temporaryDirectory, "home")
	stateDirectory := filepath.Join(temporaryDirectory, "state")
	pidPath := filepath.Join(temporaryDirectory, "server.pid")
	if err := os.MkdirAll(homeDirectory, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v, want nil", err)
	}

	executable := writeDaemonExecutable(t, temporaryDirectory, `#!/bin/sh
if [ "$WADE_INTERNAL_SERVER_READY_FD" != "3" ]; then
  printf '{"error":"unexpected readiness file descriptor"}\n' >&3
  exit 1
fi
if [ "$1" != "test-daemon" ]; then
  printf '{"error":"unexpected daemon argument"}\n' >&3
  exit 1
fi
printf '%s' "$$" > "$WADE_TEST_PID_PATH"
printf '{"status":{"address":"test.localhost:1234","pid":%s,"logPath":"%s/wade/server.log"}}\n' "$$" "$XDG_STATE_HOME" >&3
exec sleep 30
`)
	t.Setenv("HOME", homeDirectory)
	t.Setenv("XDG_STATE_HOME", stateDirectory)
	t.Setenv("WADE_TEST_PID_PATH", pidPath)
	t.Setenv(serverReadyFileEnv, "99")

	manager := NewManager()
	manager.executablePath = func() (string, error) { return executable, nil }
	status, err := manager.Start(testDaemonArgument)
	if err != nil {
		t.Fatalf("Start() error = %v, want nil", err)
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
	t.Cleanup(func() { _ = process.Signal(syscall.SIGTERM) })

	if status.PID != pid || status.Address != "test.localhost:1234" {
		t.Fatalf("Start() status = %#v, want PID %d and test address", status, pid)
	}
	wantLogPath := filepath.Join(stateDirectory, "wade", "server.log")
	if status.LogPath != wantLogPath {
		t.Fatalf("Start() LogPath = %q, want %q", status.LogPath, wantLogPath)
	}
	if err := process.Signal(syscall.Signal(0)); err != nil {
		t.Fatalf("background process is not running: %v", err)
	}
}

func TestManagerReportsBackgroundStartupFailure(t *testing.T) {
	temporaryDirectory := shortStateRoot(t)
	homeDirectory := filepath.Join(temporaryDirectory, "home")
	if err := os.MkdirAll(homeDirectory, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v, want nil", err)
	}

	executable := writeDaemonExecutable(t, temporaryDirectory, `#!/bin/sh
printf '{"error":"address is already in use"}\n' >&3
exec sleep 30
`)
	t.Setenv("HOME", homeDirectory)
	t.Setenv("XDG_STATE_HOME", filepath.Join(temporaryDirectory, "state"))

	manager := NewManager()
	manager.executablePath = func() (string, error) { return executable, nil }
	_, err := manager.Start(testDaemonArgument)
	var startupError StartupError
	if !errors.As(err, &startupError) || !strings.Contains(err.Error(), "address is already in use") {
		t.Fatalf("Start() error = %v, want startup failure", err)
	}
}

func TestManagerTimesOutBackgroundStartup(t *testing.T) {
	temporaryDirectory := shortStateRoot(t)
	homeDirectory := filepath.Join(temporaryDirectory, "home")
	if err := os.MkdirAll(homeDirectory, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v, want nil", err)
	}

	executable := writeDaemonExecutable(t, temporaryDirectory, "#!/bin/sh\nexec sleep 30\n")
	t.Setenv("HOME", homeDirectory)
	t.Setenv("XDG_STATE_HOME", filepath.Join(temporaryDirectory, "state"))

	manager := NewManager()
	manager.executablePath = func() (string, error) { return executable, nil }
	manager.startupTimeout = 50 * time.Millisecond
	_, err := manager.Start(testDaemonArgument)
	var startupError StartupError
	if !errors.As(err, &startupError) || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("Start() error = %v, want startup timeout", err)
	}
}

func TestConsumeStartupReporterClearsEnvironment(t *testing.T) {
	if os.Getenv("WADE_TEST_CONSUME_REPORTER") == "1" {
		reporter, err := ConsumeStartupReporter()
		if err != nil || reporter == nil {
			os.Exit(2)
		}
		if _, found := os.LookupEnv(serverReadyFileEnv); found {
			os.Exit(3)
		}
		if err := reporter.ReportReady(Status{PID: 12345, Address: "test.localhost:1234", LogPath: "/tmp/server.log"}); err != nil {
			os.Exit(4)
		}
		os.Exit(0)
	}

	readyReader, readyWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe() error = %v, want nil", err)
	}
	defer readyReader.Close()

	command := exec.Command(os.Args[0], "-test.run=TestConsumeStartupReporterClearsEnvironment")
	command.ExtraFiles = []*os.File{readyWriter}
	command.Env = append(os.Environ(), "WADE_TEST_CONSUME_REPORTER=1", serverReadyFileEnv+"=3")
	if err := command.Start(); err != nil {
		t.Fatalf("Start() error = %v, want nil", err)
	}
	_ = readyWriter.Close()

	var message startupMessage
	if err := json.NewDecoder(readyReader).Decode(&message); err != nil {
		t.Fatalf("Decode() error = %v, want nil", err)
	}
	if err := command.Wait(); err != nil {
		t.Fatalf("helper process error = %v, want nil", err)
	}
	if message.Status == nil || message.Status.PID != 12345 {
		t.Fatalf("startup message = %#v, want reported status", message)
	}
}

func TestStartupReporterIsConsumedAfterReportFailure(t *testing.T) {
	readyReader, readyWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe() error = %v, want nil", err)
	}
	if err := readyReader.Close(); err != nil {
		t.Fatalf("reader Close() error = %v, want nil", err)
	}
	t.Cleanup(func() { _ = readyWriter.Close() })

	reporter := &StartupReporter{file: readyWriter}
	err = reporter.ReportReady(Status{PID: 12345, Address: "test.localhost:1234", LogPath: "/tmp/server.log"})
	if err == nil {
		t.Fatal("ReportReady() error = nil, want closed pipe error")
	}
	if reporter.file != nil {
		t.Fatal("ReportReady() retained readiness file after failure")
	}

	reporter.Close(errors.New("startup failed"))
}

func TestBackgroundEnvironmentReplacesReadinessDescriptor(t *testing.T) {
	t.Setenv(serverReadyFileEnv, "99")

	environment := backgroundEnvironment()
	matches := make([]string, 0, 1)
	for _, entry := range environment {
		if strings.HasPrefix(entry, serverReadyFileEnv+"=") {
			matches = append(matches, entry)
		}
	}
	if len(matches) != 1 || matches[0] != serverReadyFileEnv+"=3" {
		t.Fatalf("readiness environment = %#v, want one descriptor 3 entry", matches)
	}
}

func writeDaemonExecutable(t *testing.T, directory string, contents string) string {
	t.Helper()

	path := filepath.Join(directory, "wade-test-daemon")
	if err := os.WriteFile(path, []byte(contents), 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v, want nil", err)
	}
	return path
}
