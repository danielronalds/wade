package sessions

import (
	"errors"
	"reflect"
	"testing"
)

type projectServiceStub struct {
	path    string
	pathErr error
}

func (s projectServiceStub) Path(string) (string, error) {
	return s.path, s.pathErr
}

func (projectServiceStub) NamesForDirectories([]string) []string {
	return nil
}

type terminalSessionServiceStub struct {
	activeAgentSessions int
	writeErr            error
	directory           string
	data                []byte
}

func (s *terminalSessionServiceStub) ActiveDirectories() []string {
	return nil
}

func (s *terminalSessionServiceStub) CloseSessionsForDirectory(string) int {
	return 0
}

func (s *terminalSessionServiceStub) WriteToActiveAgent(directory string, data []byte) (int, error) {
	s.directory = directory
	s.data = append([]byte(nil), data...)
	return s.activeAgentSessions, s.writeErr
}

func TestSendToAgentWritesBracketedPasteWithoutTrimmingText(t *testing.T) {
	terminals := &terminalSessionServiceStub{activeAgentSessions: 1}
	service := NewService(projectServiceStub{path: "/projects/wade"}, terminals)

	err := service.SendToAgent("wade", "  @main.go:10\n ")
	if err != nil {
		t.Fatalf("SendToAgent() error = %v", err)
	}

	if terminals.directory != "/projects/wade" {
		t.Fatalf("WriteToActiveAgent() directory = %q, want %q", terminals.directory, "/projects/wade")
	}

	want := []byte("\x1b[200~  @main.go:10\n \x1b[201~")
	if !reflect.DeepEqual(terminals.data, want) {
		t.Fatalf("WriteToActiveAgent() data = %q, want %q", terminals.data, want)
	}
}

func TestSendToAgentReturnsAgentSessionErrors(t *testing.T) {
	tests := map[string]struct {
		activeAgentSessions int
		wantErr             error
	}{
		"no active agent": {
			activeAgentSessions: 0,
			wantErr:             ErrAgentSessionNotFound,
		},
		"ambiguous active agents": {
			activeAgentSessions: 2,
			wantErr:             ErrAgentSessionAmbiguous,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			terminals := &terminalSessionServiceStub{activeAgentSessions: test.activeAgentSessions}
			service := NewService(projectServiceStub{path: "/projects/wade"}, terminals)

			err := service.SendToAgent("wade", "reference")
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("SendToAgent() error = %v, want %v", err, test.wantErr)
			}
		})
	}
}

func TestSendToAgentReturnsSessionNotFoundForUnknownProject(t *testing.T) {
	terminals := &terminalSessionServiceStub{activeAgentSessions: 1}
	service := NewService(projectServiceStub{pathErr: errors.New("project not found")}, terminals)

	err := service.SendToAgent("unknown", "reference")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("SendToAgent() error = %v, want %v", err, ErrSessionNotFound)
	}
}

func TestSendToAgentReturnsTerminalWriteError(t *testing.T) {
	writeErr := errors.New("write failed")
	terminals := &terminalSessionServiceStub{activeAgentSessions: 1, writeErr: writeErr}
	service := NewService(projectServiceStub{path: "/projects/wade"}, terminals)

	err := service.SendToAgent("wade", "reference")
	if !errors.Is(err, writeErr) {
		t.Fatalf("SendToAgent() error = %v, want %v", err, writeErr)
	}
}
