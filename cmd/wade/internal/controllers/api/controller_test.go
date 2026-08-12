// This file is completely vibecoded and needs to be properly reviewed.

package api

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"wade/internal/daemon"
)

type environmentStub map[string]string

func (e environmentStub) Variable(name string) string {
	return e[name]
}

type daemonStub struct {
	status daemon.Status
	err    error
}

func (d daemonStub) Status() (daemon.Status, error) {
	return d.status, d.err
}

func newTestController(stdout io.Writer, stdin io.Reader) Controller {
	return Controller{
		stdout:      stdout,
		stdin:       stdin,
		environment: environmentStub{},
		daemon:      daemonStub{err: errors.New("not running")},
		httpClient:  &http.Client{},
	}
}

func TestHandleArgsWritesCommandList(t *testing.T) {
	for _, arguments := range [][]string{{"api"}, {"api", "--help"}, {"api", "-h"}, {"api", "help"}} {
		t.Run(strings.Join(arguments, " "), func(t *testing.T) {
			var output bytes.Buffer
			controller := newTestController(&output, strings.NewReader(""))

			exitCode, err := controller.HandleArgs(arguments)
			if err != nil {
				t.Fatalf("HandleArgs() error = %v, want nil", err)
			}
			if exitCode != 0 {
				t.Fatalf("HandleArgs() exit code = %d, want 0", exitCode)
			}
			if !strings.Contains(output.String(), "list-workspaces") {
				t.Fatalf("output %q does not list list-workspaces", output.String())
			}
			if strings.Contains(output.String(), "connect-workspace-terminal") {
				t.Fatalf("output %q lists the excluded connect-workspace-terminal", output.String())
			}
		})
	}
}

func TestHandleArgsWritesOperationHelp(t *testing.T) {
	var output bytes.Buffer
	controller := newTestController(&output, strings.NewReader(""))

	exitCode, err := controller.HandleArgs([]string{"api", "put-workspace-terminal", "--help"})
	if err != nil {
		t.Fatalf("HandleArgs() error = %v, want nil", err)
	}
	if exitCode != 0 {
		t.Fatalf("HandleArgs() exit code = %d, want 0", exitCode)
	}

	help := output.String()
	for _, expected := range []string{
		"Usage: wade api put-workspace-terminal [flags]",
		"PUT /api/v1/workspaces/{workspaceId}/terminals/{terminalId}",
		"--workspace-id",
		"--terminal-id",
		"--address",
		"(required)",
	} {
		if !strings.Contains(help, expected) {
			t.Fatalf("help %q does not contain %q", help, expected)
		}
	}
	if strings.Contains(help, "--body") {
		t.Fatalf("help %q offers --body for a bodyless operation", help)
	}
}

func TestHandleArgsReturnsUnknownCommandError(t *testing.T) {
	controller := newTestController(&bytes.Buffer{}, strings.NewReader(""))

	_, err := controller.HandleArgs([]string{"api", "list-nothing"})
	if err == nil || !strings.Contains(err.Error(), "unknown api command: list-nothing") {
		t.Fatalf("HandleArgs() error = %v, want unknown api command error", err)
	}
}

func TestResolveAddressPrecedence(t *testing.T) {
	tests := []struct {
		name        string
		environment environmentStub
		daemon      daemonStub
		want        string
	}{
		{
			name:        "development mode wins over address and daemon",
			environment: environmentStub{"WADE_DEV": "1", "WADE_ADDR": "configured:1111"},
			daemon:      daemonStub{status: daemon.Status{Address: "daemon:2222"}},
			want:        "editor-dev.localhost:8090",
		},
		{
			name:        "environment address wins over daemon",
			environment: environmentStub{"WADE_ADDR": "configured:1111"},
			daemon:      daemonStub{status: daemon.Status{Address: "daemon:2222"}},
			want:        "configured:1111",
		},
		{
			name:        "disabled development mode is ignored",
			environment: environmentStub{"WADE_DEV": "false", "WADE_ADDR": "configured:1111"},
			daemon:      daemonStub{err: errors.New("not running")},
			want:        "configured:1111",
		},
		{
			name:        "daemon address wins over default",
			environment: environmentStub{},
			daemon:      daemonStub{status: daemon.Status{Address: "daemon:2222"}},
			want:        "daemon:2222",
		},
		{
			name:        "default address when nothing is configured",
			environment: environmentStub{},
			daemon:      daemonStub{err: errors.New("not running")},
			want:        "editor.localhost:8765",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			controller := newTestController(&bytes.Buffer{}, strings.NewReader(""))
			controller.environment = test.environment
			controller.daemon = test.daemon

			if got := controller.resolveAddress(); got != test.want {
				t.Fatalf("resolveAddress() = %q, want %q", got, test.want)
			}
		})
	}
}
