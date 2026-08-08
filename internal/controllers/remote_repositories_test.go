package controllers

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"wade/internal/models/remoterepositories"
)

func TestRemoteRepositoryListUsesOneBulkLocalMapping(t *testing.T) {
	repositoryModel := &fakeRepositoriesModel{workspaceIDsByRemote: map[string][]string{"example/wade": {"wade"}}}
	handler := NewRemoteRepositories(fakeRemoteRepositoriesModel{items: []remoterepositories.RemoteRepository{{ID: "example/wade"}}}, repositoryModel)
	response := httptest.NewRecorder()

	handler.List(response, httptest.NewRequest(http.MethodGet, "/api/v1/remote-repositories", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d", response.Code)
	}
	if repositoryModel.workspaceMappingCalls != 1 {
		t.Fatalf("mapping calls = %d, want 1", repositoryModel.workspaceMappingCalls)
	}
	if !reflect.DeepEqual(repositoryModel.workspaceIDsByRemote["example/wade"], []string{"wade"}) {
		t.Fatalf("workspace mapping = %#v", repositoryModel.workspaceIDsByRemote)
	}
}
