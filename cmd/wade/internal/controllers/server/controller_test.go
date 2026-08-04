package server

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"wade/internal/daemon"
)

type daemonStub struct {
	startStatus daemon.Status
	status      daemon.Status
	startError  error
	statusError error
	stopError   error
}

func (s daemonStub) Acquire(string) (*daemon.ControlServer, error) {
	return nil, errors.New("unexpected Acquire() call")
}

func (s daemonStub) Start() (daemon.Status, error) {
	return s.startStatus, s.startError
}

func (s daemonStub) Status() (daemon.Status, error) {
	return s.status, s.statusError
}

func (s daemonStub) Stop() error {
	return s.stopError
}

func TestControllerStartsBackgroundDaemon(t *testing.T) {
	status := daemon.Status{
		PID:     12345,
		Address: "test.localhost:1234",
		LogPath: "/tmp/wade/server.log",
	}
	var output bytes.Buffer
	controller := Controller{stdout: &output, daemon: daemonStub{startStatus: status}}

	exitCode, err := controller.HandleArgs([]string{"server"})
	if err != nil {
		t.Fatalf("HandleArgs() error = %v, want nil", err)
	}
	if exitCode != 0 {
		t.Fatalf("HandleArgs() exit code = %d, want 0", exitCode)
	}

	want := "WADE server listening on test.localhost:1234\nPID: 12345\nLog: /tmp/wade/server.log\n"
	if output.String() != want {
		t.Fatalf("output = %q, want %q", output.String(), want)
	}
}

func TestControllerReportsExistingDaemon(t *testing.T) {
	status := daemon.Status{PID: 12345, Address: "test.localhost:1234", LogPath: "/tmp/server.log"}
	var output bytes.Buffer
	controller := Controller{
		stdout: &output,
		daemon: daemonStub{startError: daemon.AlreadyRunningError{Status: status}},
	}

	exitCode, err := controller.HandleArgs([]string{"server"})
	if err != nil {
		t.Fatalf("HandleArgs() error = %v, want nil", err)
	}
	if exitCode != 0 {
		t.Fatalf("HandleArgs() exit code = %d, want 0", exitCode)
	}

	want := "WADE is already running\nPID: 12345\nAddress: test.localhost:1234\n"
	if output.String() != want {
		t.Fatalf("output = %q, want %q", output.String(), want)
	}
}

func TestControllerReportsRunningStatus(t *testing.T) {
	status := daemon.Status{
		PID:     12345,
		Address: "test.localhost:1234",
		LogPath: "/tmp/wade/server.log",
	}
	var output bytes.Buffer
	controller := Controller{stdout: &output, daemon: daemonStub{status: status}}

	exitCode, err := controller.HandleArgs([]string{"status"})
	if err != nil {
		t.Fatalf("HandleArgs() error = %v, want nil", err)
	}
	if exitCode != 0 {
		t.Fatalf("HandleArgs() exit code = %d, want 0", exitCode)
	}

	want := "WADE is running\nPID: 12345\nAddress: test.localhost:1234\nLog: /tmp/wade/server.log\n"
	if output.String() != want {
		t.Fatalf("output = %q, want %q", output.String(), want)
	}
}

func TestControllerReturnsNonZeroStatusWhenStopped(t *testing.T) {
	var output bytes.Buffer
	controller := Controller{
		stdout: &output,
		daemon: daemonStub{statusError: daemon.NotRunningError{}},
	}

	exitCode, err := controller.HandleArgs([]string{"status"})
	if err != nil {
		t.Fatalf("HandleArgs() error = %v, want nil", err)
	}
	if exitCode != 1 {
		t.Fatalf("HandleArgs() exit code = %d, want 1", exitCode)
	}
	if output.String() != "WADE is not running\n" {
		t.Fatalf("output = %q, want stopped output", output.String())
	}
}

func TestControllerStopsDaemon(t *testing.T) {
	var output bytes.Buffer
	controller := Controller{stdout: &output, daemon: daemonStub{}}

	exitCode, err := controller.HandleArgs([]string{"stop"})
	if err != nil {
		t.Fatalf("HandleArgs() error = %v, want nil", err)
	}
	if exitCode != 0 || output.String() != "WADE stopped\n" {
		t.Fatalf("HandleArgs() = (%d, %q), want successful stop", exitCode, output.String())
	}
}

func TestControllerStopIsIdempotent(t *testing.T) {
	var output bytes.Buffer
	controller := Controller{
		stdout: &output,
		daemon: daemonStub{stopError: daemon.NotRunningError{}},
	}

	exitCode, err := controller.HandleArgs([]string{"stop"})
	if err != nil {
		t.Fatalf("HandleArgs() error = %v, want nil", err)
	}
	if exitCode != 0 || output.String() != "WADE is not running\n" {
		t.Fatalf("HandleArgs() = (%d, %q), want idempotent stop", exitCode, output.String())
	}
}

func TestControllerReturnsDaemonErrors(t *testing.T) {
	wantError := errors.New("control failure")
	controller := Controller{
		stdout: &bytes.Buffer{},
		daemon: daemonStub{statusError: wantError},
	}

	_, err := controller.HandleArgs([]string{"status"})
	if !errors.Is(err, wantError) {
		t.Fatalf("HandleArgs() error = %v, want %v", err, wantError)
	}
}

func TestControllerRejectsUnexpectedArguments(t *testing.T) {
	tests := []struct {
		args      []string
		wantError string
	}{
		{args: []string{"server", "unexpected"}, wantError: "usage: wade server [--foreground]"},
		{args: []string{"status", "unexpected"}, wantError: "usage: wade status"},
		{args: []string{"stop", "unexpected"}, wantError: "usage: wade stop"},
	}

	for _, test := range tests {
		t.Run(strings.Join(test.args, "_"), func(t *testing.T) {
			controller := Controller{stdout: &bytes.Buffer{}, daemon: daemonStub{}}

			_, err := controller.HandleArgs(test.args)
			if err == nil || err.Error() != test.wantError {
				t.Fatalf("HandleArgs() error = %v, want %q", err, test.wantError)
			}
		})
	}
}

func TestControllerReturnsStartupFailure(t *testing.T) {
	wantError := daemon.StartupError{Message: "address is already in use", LogPath: "/tmp/server.log"}
	controller := Controller{
		stdout: &bytes.Buffer{},
		daemon: daemonStub{startError: wantError},
	}

	_, err := controller.HandleArgs([]string{"server"})
	if err == nil || !strings.Contains(err.Error(), fmt.Sprint(wantError)) {
		t.Fatalf("HandleArgs() error = %v, want startup failure", err)
	}
}
