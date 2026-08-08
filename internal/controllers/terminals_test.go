package controllers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"wade/internal/models/terminals"
)

func TestPutTerminalReturnsCreatedThenOK(t *testing.T) {
	terminalModel := &fakeTerminalsModel{
		putItem:    terminals.Terminal{ID: "misc", WorkspaceID: "wade"},
		putCreated: true,
	}
	handler := NewTerminals(terminalModel, func(*http.Request) bool { return true })

	firstResponse := httptest.NewRecorder()
	handler.Put(firstResponse, terminalRequest(http.MethodPut, "misc"))
	if firstResponse.Code != http.StatusCreated {
		t.Fatalf("first PUT status = %d", firstResponse.Code)
	}
	if location := firstResponse.Header().Get("Location"); location != "/api/v1/workspaces/wade/terminals/misc" {
		t.Fatalf("Location = %q", location)
	}

	terminalModel.putCreated = false
	secondResponse := httptest.NewRecorder()
	handler.Put(secondResponse, terminalRequest(http.MethodPut, "misc"))
	if secondResponse.Code != http.StatusOK {
		t.Fatalf("second PUT status = %d", secondResponse.Code)
	}
}

func TestConnectTerminalDoesNotCreateMissingTerminal(t *testing.T) {
	terminalModel := &fakeTerminalsModel{connectError: terminals.TerminalNotFoundError{WorkspaceID: "wade", TerminalID: "misc"}}
	handler := NewTerminals(terminalModel, func(*http.Request) bool { return true })
	response := httptest.NewRecorder()

	handler.Connect(response, terminalRequest(http.MethodGet, "misc"))

	if response.Code != http.StatusNotFound {
		t.Fatalf("socket status = %d", response.Code)
	}
	if terminalModel.putCalls != 0 {
		t.Fatalf("Put() calls = %d, want 0", terminalModel.putCalls)
	}
}

func terminalRequest(method string, terminalID string) *http.Request {
	request := httptest.NewRequest(method, "/api/v1/workspaces/wade/terminals/"+terminalID, nil)
	request.SetPathValue("workspaceId", "wade")
	request.SetPathValue("terminalId", terminalID)
	return request
}
