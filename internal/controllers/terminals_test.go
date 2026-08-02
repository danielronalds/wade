package controllers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	terminalservice "wade/internal/services/terminals"
)

type terminalWorkspaceStub struct {
	path string
}

func (s terminalWorkspaceStub) Path(string) (string, error) {
	return s.path, nil
}

func (terminalWorkspaceStub) IDForDirectory(string) (string, bool, error) {
	return "", false, nil
}

func TestPutTerminalReturnsCreatedThenOK(t *testing.T) {
	service := terminalservice.NewService(
		terminalWorkspaceStub{path: t.TempDir()},
		"/bin/sh",
		"editor.localhost:8765",
		nil,
	)
	t.Cleanup(service.Close)
	handler := NewTerminals(service, func(*http.Request) bool { return true })

	firstResponse := httptest.NewRecorder()
	handler.Put(firstResponse, terminalRequest(http.MethodPut, "misc"))
	if firstResponse.Code != http.StatusCreated {
		t.Fatalf("first PUT status = %d, want %d", firstResponse.Code, http.StatusCreated)
	}
	if location := firstResponse.Header().Get("Location"); location != "/api/v1/workspaces/wade/terminals/misc" {
		t.Fatalf("Location = %q, want terminal resource URL", location)
	}

	secondResponse := httptest.NewRecorder()
	handler.Put(secondResponse, terminalRequest(http.MethodPut, "misc"))
	if secondResponse.Code != http.StatusOK {
		t.Fatalf("second PUT status = %d, want %d", secondResponse.Code, http.StatusOK)
	}
}

func TestConnectTerminalDoesNotCreateMissingTerminal(t *testing.T) {
	service := terminalservice.NewService(
		terminalWorkspaceStub{path: t.TempDir()},
		"/bin/sh",
		"editor.localhost:8765",
		nil,
	)
	t.Cleanup(service.Close)
	handler := NewTerminals(service, func(*http.Request) bool { return true })
	response := httptest.NewRecorder()

	handler.Connect(response, terminalRequest(http.MethodGet, "misc"))

	if response.Code != http.StatusNotFound {
		t.Fatalf("socket status = %d, want %d", response.Code, http.StatusNotFound)
	}
	if terminals, err := service.List("wade"); err != nil || len(terminals) != 0 {
		t.Fatalf("terminals after socket request = %#v, error = %v", terminals, err)
	}
}

func terminalRequest(method string, terminalID string) *http.Request {
	request := httptest.NewRequest(method, "/api/v1/workspaces/wade/terminals/"+terminalID, nil)
	request.SetPathValue("workspaceId", "wade")
	request.SetPathValue("terminalId", terminalID)
	return request
}
