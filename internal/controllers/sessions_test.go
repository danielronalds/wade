package controllers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type sessionWorkspacesStub struct {
	path string
	err  error
}

func (s sessionWorkspacesStub) Path(string) (string, error) {
	return s.path, s.err
}

type sessionTerminalsStub struct {
	activeAgentTerminals int
	writeErr             error
}

func (sessionTerminalsStub) ActiveWorkspaceIDs() []string {
	return nil
}

func (sessionTerminalsStub) DeleteAll(string) int {
	return 0
}

func (s sessionTerminalsStub) InputToSelectedAgent(string, string) (int, error) {
	return s.activeAgentTerminals, s.writeErr
}

func TestSendToAgentResponses(t *testing.T) {
	tests := map[string]struct {
		body                 string
		activeAgentTerminals int
		workspaceErr         error
		writeErr             error
		wantStatus           int
	}{
		"successful write": {
			body:                 `{"text":"@main.go:10"}`,
			activeAgentTerminals: 1,
			wantStatus:           http.StatusNoContent,
		},
		"invalid JSON": {
			body:       `{`,
			wantStatus: http.StatusBadRequest,
		},
		"empty text": {
			body:       `{"text":""}`,
			wantStatus: http.StatusBadRequest,
		},
		"no active agent": {
			body:       `{"text":"reference"}`,
			wantStatus: http.StatusNotFound,
		},
		"ambiguous agents": {
			body:                 `{"text":"reference"}`,
			activeAgentTerminals: 2,
			wantStatus:           http.StatusConflict,
		},
		"unknown workspace": {
			body:         `{"text":"reference"}`,
			workspaceErr: errors.New("workspace not found"),
			wantStatus:   http.StatusNotFound,
		},
		"terminal write failure": {
			body:                 `{"text":"reference"}`,
			activeAgentTerminals: 1,
			writeErr:             errors.New("write failed"),
			wantStatus:           http.StatusInternalServerError,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			handler := NewSessions(
				sessionWorkspacesStub{path: "/workspaces/wade", err: test.workspaceErr},
				sessionTerminalsStub{
					activeAgentTerminals: test.activeAgentTerminals,
					writeErr:             test.writeErr,
				},
			)

			request := httptest.NewRequest(http.MethodPost, "/api/sessions/wade/agent", strings.NewReader(test.body))
			request.SetPathValue("projectName", "wade")
			response := httptest.NewRecorder()

			handler.SendToAgent(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("SendToAgent() status = %d, want %d", response.Code, test.wantStatus)
			}
		})
	}
}
