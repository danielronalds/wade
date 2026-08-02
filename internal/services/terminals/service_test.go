package terminals

import (
	"io"
	"os"
	"os/exec"
	"reflect"
	"sync"
	"testing"
	"time"
)

type workspaceRepositoryStub struct {
	path        string
	workspaceID string
	found       bool
}

func (s workspaceRepositoryStub) Path(string) (string, error) {
	return s.path, nil
}

func (s workspaceRepositoryStub) IDForDirectory(string) (string, bool, error) {
	return s.workspaceID, s.found, nil
}

func TestPutIsIdempotent(t *testing.T) {
	service := NewService(
		workspaceRepositoryStub{path: t.TempDir()},
		"/bin/sh",
		"editor.localhost:8765",
		nil,
	)
	t.Cleanup(service.Close)

	first, created, err := service.Put("wade", "misc")
	if err != nil {
		t.Fatalf("Put() error = %v, want nil", err)
	}
	if !created {
		t.Fatal("Put() created = false, want true")
	}
	if first.ID != "misc" || first.Role != TerminalRoleMisc || first.Status != TerminalStatusRunning {
		t.Fatalf("Put() terminal = %#v, want running misc terminal", first)
	}
	if first.SocketURL != "/api/v1/workspaces/wade/terminals/misc/socket" {
		t.Fatalf("Put() SocketURL = %q, want nested socket URL", first.SocketURL)
	}

	second, created, err := service.Put("wade", "misc")
	if err != nil {
		t.Fatalf("Put() second error = %v, want nil", err)
	}
	if created {
		t.Fatal("Put() second created = true, want false")
	}
	if first != second {
		t.Fatal("concurrent terminal identity changed")
	}
}

func TestConcurrentPutReturnsOneTerminal(t *testing.T) {
	service := NewService(
		workspaceRepositoryStub{path: t.TempDir()},
		"/bin/sh",
		"editor.localhost:8765",
		nil,
	)
	t.Cleanup(service.Close)

	const requests = 8
	terminals := make(chan *Terminal, requests)
	errors := make(chan error, requests)
	var waitGroup sync.WaitGroup
	for range requests {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			terminal, _, err := service.Put("wade", "misc")
			terminals <- terminal
			errors <- err
		}()
	}
	waitGroup.Wait()
	close(terminals)
	close(errors)

	for err := range errors {
		if err != nil {
			t.Fatalf("Put() error = %v, want nil", err)
		}
	}

	var first *Terminal
	for terminal := range terminals {
		if first == nil {
			first = terminal
			continue
		}
		if terminal != first {
			t.Fatal("concurrent Put() returned different terminals")
		}
	}
}

func TestDeleteRemovesTerminalBeforeRecreation(t *testing.T) {
	service := NewService(
		workspaceRepositoryStub{path: t.TempDir()},
		"/bin/sh",
		"editor.localhost:8765",
		nil,
	)
	t.Cleanup(service.Close)

	first, _, err := service.Put("wade", "misc")
	if err != nil {
		t.Fatalf("Put() error = %v, want nil", err)
	}
	if err := service.Delete("wade", "misc"); err != nil {
		t.Fatalf("Delete() error = %v, want nil", err)
	}
	if _, err := service.Get("wade", "misc"); err == nil {
		t.Fatal("Get() error = nil after Delete(), want not found")
	}

	second, created, err := service.Put("wade", "misc")
	if err != nil {
		t.Fatalf("replacement Put() error = %v, want nil", err)
	}
	if !created || second == first {
		t.Fatal("replacement Put() did not create a new terminal")
	}
}

func TestInputTargetsExactTerminalWithBracketedPaste(t *testing.T) {
	service := NewService(workspaceRepositoryStub{}, "/bin/sh", "editor.localhost:8765", nil)
	_, miscOutput := addWritableTerminal(t, service, "wade", "misc", TerminalRoleMisc, nil)
	_, serverOutput := addWritableTerminal(t, service, "wade", "server", TerminalRoleServer, nil)

	if err := service.Input("wade", "server", "reference", InputModeBracketedPaste); err != nil {
		t.Fatalf("Input() error = %v, want nil", err)
	}

	want := []byte("\x1b[200~reference\x1b[201~")
	got := make([]byte, len(want))
	if _, err := io.ReadFull(serverOutput, got); err != nil {
		t.Fatalf("reading server output: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("server output = %q, want %q", got, want)
	}

	if err := miscOutput.SetReadDeadline(time.Now().Add(10 * time.Millisecond)); err != nil {
		t.Fatalf("SetReadDeadline() error = %v", err)
	}
	if _, err := miscOutput.Read(make([]byte, 1)); err == nil {
		t.Fatal("misc terminal received input intended for server")
	}
}

func TestActiveWorkspaceIDsReturnsSortedUniqueWorkspaceIDs(t *testing.T) {
	service := NewService(workspaceRepositoryStub{}, "/bin/sh", "editor.localhost:8765", nil)
	_, _ = addWritableTerminal(t, service, "bravo", "misc", TerminalRoleMisc, nil)
	_, _ = addWritableTerminal(t, service, "alpha", "misc", TerminalRoleMisc, nil)
	_, _ = addWritableTerminal(t, service, "alpha", "server", TerminalRoleServer, nil)
	closedTerminal, _ := addWritableTerminal(t, service, "closed", "misc", TerminalRoleMisc, nil)
	closedTerminal.closed.Store(true)

	got := service.ActiveWorkspaceIDs()
	want := []string{"alpha", "bravo"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ActiveWorkspaceIDs() = %#v, want %#v", got, want)
	}
}

func TestCloseTerminalsForDirectoryUsesWorkspaceIdentity(t *testing.T) {
	service := NewService(
		workspaceRepositoryStub{workspaceID: "wade", found: true},
		"/bin/sh",
		"editor.localhost:8765",
		nil,
	)
	_, _ = addWritableTerminal(t, service, "wade", "misc", TerminalRoleMisc, nil)

	closed := service.CloseTerminalsForDirectory("/workspaces/wade")
	if closed != 1 {
		t.Fatalf("CloseTerminalsForDirectory() = %d, want 1", closed)
	}
	if service.ActiveTerminalCount("wade") != 0 {
		t.Fatalf("ActiveTerminalCount() = %d, want 0", service.ActiveTerminalCount("wade"))
	}
}

func addWritableTerminal(
	t *testing.T,
	service *Service,
	workspaceID string,
	terminalID string,
	role TerminalRole,
	agent *string,
) (*Terminal, *os.File) {
	t.Helper()

	outputReader, terminalFile, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating terminal pipe: %v", err)
	}
	t.Cleanup(func() {
		_ = outputReader.Close()
		_ = terminalFile.Close()
	})

	key := terminalKey(workspaceID, terminalID)
	terminal := &Terminal{
		ID:          terminalID,
		WorkspaceID: workspaceID,
		Role:        role,
		Agent:       agent,
		Status:      TerminalStatusRunning,
		key:         key,
		manager:     service,
		process:     &Process{command: &exec.Cmd{}, terminal: terminalFile},
		buffer:      newOutputBuffer(terminalBufferBytes),
		clients:     make(map[*Client]struct{}),
	}
	service.terminals[key] = terminal

	return terminal, outputReader
}
