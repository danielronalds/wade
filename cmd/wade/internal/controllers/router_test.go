package controllers

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestRouterRoutesDaemonLifecycleCommands(t *testing.T) {
	tests := []struct {
		command  string
		exitCode int
		output   string
	}{
		{command: "status", exitCode: 1, output: "WADE is not running\n"},
		{command: "stop", exitCode: 0, output: "WADE is not running\n"},
	}

	for _, test := range tests {
		t.Run(test.command, func(t *testing.T) {
			// Use /tmp directly to stay within the Unix socket path limit on macOS.
			stateRoot, err := os.MkdirTemp("/tmp", "wade-router-")
			if err != nil {
				t.Fatalf("MkdirTemp() error = %v, want nil", err)
			}
			t.Cleanup(func() { _ = os.RemoveAll(stateRoot) })
			t.Setenv("XDG_STATE_HOME", stateRoot)

			var output bytes.Buffer
			router := NewRouter(&output, nil)

			exitCode, err := router.HandleArgs([]string{test.command})
			if err != nil {
				t.Fatalf("HandleArgs() error = %v, want nil", err)
			}
			if exitCode != test.exitCode {
				t.Fatalf("HandleArgs() exit code = %d, want %d", exitCode, test.exitCode)
			}
			if output.String() != test.output {
				t.Fatalf("output = %q, want %q", output.String(), test.output)
			}
		})
	}
}

func TestRouterRoutesAPICommand(t *testing.T) {
	var output bytes.Buffer
	router := NewRouter(&output, nil)

	exitCode, err := router.HandleArgs([]string{"api"})
	if err != nil {
		t.Fatalf("HandleArgs() error = %v, want nil", err)
	}
	if exitCode != 0 {
		t.Fatalf("HandleArgs() exit code = %d, want 0", exitCode)
	}
	if !strings.Contains(output.String(), "list-workspaces") {
		t.Fatalf("output %q does not list API commands", output.String())
	}
}

func TestRouterReturnsUnknownCommandError(t *testing.T) {
	router := NewRouter(&bytes.Buffer{}, nil)

	_, err := router.HandleArgs([]string{"unknown"})
	if err == nil || !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("HandleArgs() error = %v, want unknown command error", err)
	}
}
