package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"wade/internal/models/repositories"
	"wade/internal/models/workspaces"
)

func TestListWorkspacesUsesTargetedActiveLoadingAndFilters(t *testing.T) {
	remoteRepositoryID := "example/wade"
	workspaceModel := &fakeWorkspacesModel{listByIDItems: []workspaces.WorkspaceSummary{{ID: "wade", Name: "wade"}, {ID: "notes", Name: "notes"}}}
	repositoryModel := &fakeRepositoriesModel{targetedContexts: map[string]repositories.WorkspaceContext{
		"wade": {
			WorkspaceID: "wade",
			Repository: repositories.Repository{
				ID:                 "wade",
				RemoteRepositoryID: &remoteRepositoryID,
			},
		},
	}}
	terminalModel := &fakeTerminalsModel{
		activeWorkspaceIDs: []string{"wade", "notes"},
		activeCounts:       map[string]int{"wade": 1},
	}
	handler := NewWorkspaces(workspaceModel, repositoryModel, terminalModel)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces?activity=active&repositoryId=wade&remoteRepositoryId=example%2Fwade", nil)
	response := httptest.NewRecorder()

	handler.List(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d", response.Code)
	}
	var body WorkspaceList
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 1 || body.Items[0].ID != "wade" {
		t.Fatalf("items = %#v", body.Items)
	}
	if !reflect.DeepEqual(repositoryModel.targetedWorkspaceIDCall, []string{"wade", "notes"}) {
		t.Fatalf("targeted context IDs = %#v", repositoryModel.targetedWorkspaceIDCall)
	}
}

func TestMaterialiseWorkspaceReturnsCreatedResourceAndLocation(t *testing.T) {
	var materialiseRequest workspaces.MaterialiseRequest
	handler := NewWorkspaces(
		&fakeWorkspacesModel{materialised: workspaces.Workspace{ID: "wade", Name: "wade"}, materialiseRequest: &materialiseRequest},
		&fakeRepositoriesModel{},
		&fakeTerminalsModel{activeCounts: map[string]int{}},
	)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces", strings.NewReader(`{"remoteRepositoryId":"example/wade","workspaceDirectory":"~/Personal"}`))
	response := httptest.NewRecorder()

	handler.Materialise(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d", response.Code)
	}
	if location := response.Header().Get("Location"); location != "/api/v1/workspaces/wade" {
		t.Fatalf("Location = %q", location)
	}
	if materialiseRequest.RemoteRepositoryID != "example/wade" || materialiseRequest.WorkspaceDirectory != "~/Personal" {
		t.Fatalf("request = %#v", materialiseRequest)
	}
}

func TestMaterialiseWorkspaceRejectsMalformedJSON(t *testing.T) {
	handler := NewWorkspaces(&fakeWorkspacesModel{}, &fakeRepositoriesModel{}, &fakeTerminalsModel{})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces", strings.NewReader(`{`))
	response := httptest.NewRecorder()

	handler.Materialise(response, request)

	if response.Code != http.StatusBadRequest || response.Header().Get("Content-Type") != "application/problem+json" {
		t.Fatalf("status/content type = %d/%q", response.Code, response.Header().Get("Content-Type"))
	}
}
