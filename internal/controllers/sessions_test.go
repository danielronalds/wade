package controllers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"wade/internal/services/sessions"
)

type sessionProjectsStub struct {
	path string
	err  error
}

func (s sessionProjectsStub) Path(string) (string, error) {
	return s.path, s.err
}

func (sessionProjectsStub) NamesForDirectories([]string) []string {
	return nil
}

type sessionTerminalsStub struct {
	activeAgentSessions int
	writeErr            error
}

func (sessionTerminalsStub) ActiveDirectories() []string {
	return nil
}

func (sessionTerminalsStub) CloseSessionsForDirectory(string) int {
	return 0
}

func (s sessionTerminalsStub) WriteToActiveAgent(string, []byte) (int, error) {
	return s.activeAgentSessions, s.writeErr
}

func TestSendToAgentResponses(t *testing.T) {
	tests := map[string]struct {
		body                string
		activeAgentSessions int
		projectErr          error
		writeErr            error
		wantStatus          int
	}{
		"successful write": {
			body:                `{"text":"@main.go:10"}`,
			activeAgentSessions: 1,
			wantStatus:          http.StatusNoContent,
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
			body:                `{"text":"reference"}`,
			activeAgentSessions: 2,
			wantStatus:          http.StatusConflict,
		},
		"unknown project": {
			body:       `{"text":"reference"}`,
			projectErr: errors.New("project not found"),
			wantStatus: http.StatusNotFound,
		},
		"terminal write failure": {
			body:                `{"text":"reference"}`,
			activeAgentSessions: 1,
			writeErr:            errors.New("write failed"),
			wantStatus:          http.StatusInternalServerError,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			service := sessions.NewService(
				sessionProjectsStub{path: "/projects/wade", err: test.projectErr},
				sessionTerminalsStub{
					activeAgentSessions: test.activeAgentSessions,
					writeErr:            test.writeErr,
				},
			)
			handler := NewSessions(service)

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
