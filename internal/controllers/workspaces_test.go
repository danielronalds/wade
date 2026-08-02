package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"wade/internal/services/remoterepositories"
	"wade/internal/services/workspaces"
)

type workspaceServiceStub struct {
	items       []workspaces.WorkspaceSummary
	activeItems []workspaces.WorkspaceSummary
}

func (s workspaceServiceStub) List(context.Context) ([]workspaces.WorkspaceSummary, error) {
	return s.items, nil
}

func (s workspaceServiceStub) ListActive(context.Context) ([]workspaces.WorkspaceSummary, error) {
	return s.activeItems, nil
}

func (workspaceServiceStub) Get(context.Context, string) (workspaces.Workspace, error) {
	return workspaces.Workspace{}, nil
}

type workspaceMaterialiserStub struct {
	workspace workspaces.Workspace
	request   *remoterepositories.CloneRequest
}

func (s workspaceMaterialiserStub) Clone(_ context.Context, request remoterepositories.CloneRequest) (workspaces.Workspace, error) {
	*s.request = request
	return s.workspace, nil
}

func TestListWorkspacesFiltersUsingV1QueryParameters(t *testing.T) {
	repositoryID := "wade"
	remoteRepositoryID := "example/wade"
	handler := NewWorkspaces(workspaceServiceStub{activeItems: []workspaces.WorkspaceSummary{
		{
			ID:                 "wade",
			RepositoryID:       &repositoryID,
			RemoteRepositoryID: &remoteRepositoryID,
			Activity:           workspaces.WorkspaceActivity{ActiveTerminalCount: 1},
		},
		{ID: "notes"},
	}}, nil)
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/workspaces?activity=active&repositoryId=wade&remoteRepositoryId=example%2Fwade",
		nil,
	)
	response := httptest.NewRecorder()

	handler.List(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	var body WorkspaceList
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if len(body.Items) != 1 || body.Items[0].ID != "wade" {
		t.Fatalf("items = %#v, want only wade", body.Items)
	}
}

func TestMaterialiseWorkspaceReturnsCreatedResourceAndLocation(t *testing.T) {
	var cloneRequest remoterepositories.CloneRequest
	handler := NewWorkspaces(workspaceServiceStub{}, workspaceMaterialiserStub{
		workspace: workspaces.Workspace{ID: "wade", Name: "wade"},
		request:   &cloneRequest,
	})
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/workspaces",
		strings.NewReader(`{"remoteRepositoryId":"example/wade","workspaceDirectory":"~/Personal"}`),
	)
	response := httptest.NewRecorder()

	handler.Materialise(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusCreated)
	}
	if location := response.Header().Get("Location"); location != "/api/v1/workspaces/wade" {
		t.Fatalf("Location = %q, want /api/v1/workspaces/wade", location)
	}
	if cloneRequest.RemoteRepositoryID != "example/wade" || cloneRequest.WorkspaceDirectory != "~/Personal" {
		t.Fatalf("clone request = %#v", cloneRequest)
	}
}

func TestMaterialiseWorkspaceRejectsMalformedJSON(t *testing.T) {
	handler := NewWorkspaces(workspaceServiceStub{}, nil)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces", strings.NewReader(`{`))
	response := httptest.NewRecorder()

	handler.Materialise(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "application/problem+json" {
		t.Fatalf("Content-Type = %q, want application/problem+json", contentType)
	}
}
