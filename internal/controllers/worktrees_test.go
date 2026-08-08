package controllers

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"wade/internal/models/repositories"
)

func TestDeleteWorktreeInspectsClosesThenRemoves(t *testing.T) {
	calls := make([]string, 0)
	repositoryModel := &fakeRepositoriesModel{
		worktree: repositories.Worktree{ID: "wade-feature", WorkspaceID: "wade-feature", IsRemovable: true},
		calls:    &calls,
	}
	terminalModel := &fakeTerminalsModel{calls: &calls}
	handler := NewWorktrees(repositoryModel, terminalModel)
	request := httptest.NewRequest(http.MethodDelete, "/api/v1/repositories/wade/worktrees/wade-feature", nil)
	request.SetPathValue("repositoryId", "wade")
	request.SetPathValue("worktreeId", "wade-feature")
	response := httptest.NewRecorder()

	handler.Delete(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d", response.Code)
	}
	if !reflect.DeepEqual(calls, []string{"get", "close", "remove"}) {
		t.Fatalf("calls = %#v", calls)
	}
	if !reflect.DeepEqual(terminalModel.deleteAllCalls, []string{"wade-feature"}) {
		t.Fatalf("closed workspaces = %#v", terminalModel.deleteAllCalls)
	}
}
