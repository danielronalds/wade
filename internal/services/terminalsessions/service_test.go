package terminalsessions

import (
	"io"
	"os"
	"reflect"
	"testing"
)

func TestWriteToActiveAgentUsesSelectedAgentSession(t *testing.T) {
	service := NewService("/bin/sh", "editor.localhost:8765", nil)
	_, _ = addWritableProjectSession(t, service, "/projects/wade", agentTerminalName, "pi")
	selectedSession, selectedOutput := addWritableProjectSession(t, service, "/projects/wade", agentTerminalName, "claude")
	selectedSession.ApplyControlMessage([]byte(`{"type":"activate"}`))

	input := []byte("reference")
	activeSessions, err := service.WriteToActiveAgent("/projects/wade", input)
	if err != nil {
		t.Fatalf("WriteToActiveAgent() error = %v", err)
	}
	if activeSessions != 1 {
		t.Fatalf("WriteToActiveAgent() active sessions = %d, want 1", activeSessions)
	}

	output := make([]byte, len(input))
	if _, err := io.ReadFull(selectedOutput, output); err != nil {
		t.Fatalf("reading selected agent output: %v", err)
	}
	if !reflect.DeepEqual(output, input) {
		t.Fatalf("selected agent output = %q, want %q", output, input)
	}
}

func TestWriteToActiveAgentUsesSoleAgentWithoutSelection(t *testing.T) {
	service := NewService("/bin/sh", "editor.localhost:8765", nil)
	_, outputReader := addWritableProjectSession(t, service, "/projects/wade", agentTerminalName, "pi")
	_, _ = addWritableProjectSession(t, service, "/projects/wade", "misc", "")

	input := []byte("reference")
	activeSessions, err := service.WriteToActiveAgent("/projects/wade", input)
	if err != nil {
		t.Fatalf("WriteToActiveAgent() error = %v", err)
	}
	if activeSessions != 1 {
		t.Fatalf("WriteToActiveAgent() active sessions = %d, want 1", activeSessions)
	}

	output := make([]byte, len(input))
	if _, err := io.ReadFull(outputReader, output); err != nil {
		t.Fatalf("reading agent output: %v", err)
	}
	if !reflect.DeepEqual(output, input) {
		t.Fatalf("agent output = %q, want %q", output, input)
	}
}

func TestWriteToActiveAgentReportsAmbiguousAgentsWithoutSelection(t *testing.T) {
	service := NewService("/bin/sh", "editor.localhost:8765", nil)
	_, _ = addWritableProjectSession(t, service, "/projects/wade", agentTerminalName, "pi")
	_, _ = addWritableProjectSession(t, service, "/projects/wade", agentTerminalName, "claude")

	activeSessions, err := service.WriteToActiveAgent("/projects/wade", []byte("reference"))
	if err != nil {
		t.Fatalf("WriteToActiveAgent() error = %v", err)
	}
	if activeSessions != 2 {
		t.Fatalf("WriteToActiveAgent() active sessions = %d, want 2", activeSessions)
	}
}

func TestAgentActivationIgnoresNonAgentTerminal(t *testing.T) {
	service := NewService("/bin/sh", "editor.localhost:8765", nil)
	miscSession, _ := addWritableProjectSession(t, service, "/projects/wade", "misc", "")
	miscSession.ApplyControlMessage([]byte(`{"type":"activate"}`))

	if service.selectedAgentSessions["/projects/wade"] != nil {
		t.Fatal("expected misc terminal activation to be ignored")
	}
}

func addWritableProjectSession(
	t *testing.T,
	service *Service,
	directory string,
	terminalName string,
	agentName string,
) (*ProjectSession, *os.File) {
	t.Helper()

	outputReader, terminal, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating terminal pipe: %v", err)
	}
	t.Cleanup(func() {
		_ = outputReader.Close()
		_ = terminal.Close()
	})

	key := terminalSessionKey(directory, terminalName, agentName)
	projectSession := &ProjectSession{
		key:          key,
		manager:      service,
		session:      &Session{terminal: terminal},
		terminalName: terminalName,
		directory:    directory,
	}
	service.sessions[key] = projectSession

	return projectSession, outputReader
}
