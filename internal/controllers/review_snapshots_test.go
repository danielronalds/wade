package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"wade/internal/models/reviewsnapshots"
)

func TestCreateReviewSnapshotReturnsCreatedResourceAndLocation(t *testing.T) {
	model := &fakeReviewSnapshotsModel{snapshot: reviewsnapshots.ReviewSnapshot{
		ID:          "review_snapshot_01",
		WorkspaceID: "wade",
		Files:       []reviewsnapshots.File{},
	}}
	controller := NewReviewSnapshots(model)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/wade/review-snapshots", nil)
	request.SetPathValue("workspaceId", "wade")
	response := httptest.NewRecorder()

	controller.Create(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusCreated)
	}
	if location := response.Header().Get("Location"); location != "/api/v1/review-snapshots/review_snapshot_01" {
		t.Fatalf("Location = %q", location)
	}
	if model.createdWorkspaceID != "wade" {
		t.Fatalf("Create() workspace ID = %q, want wade", model.createdWorkspaceID)
	}
}

func TestGetReviewSnapshotFileContentsDelegatesScopeValidation(t *testing.T) {
	model := &fakeReviewSnapshotsModel{contents: reviewsnapshots.FileContents{
		OriginalContent: "before",
		ModifiedContent: "after",
	}}
	controller := NewReviewSnapshots(model)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/review-snapshots/snapshot/files/file/contents?scope=working-tree", nil)
	request.SetPathValue("snapshotId", "snapshot")
	request.SetPathValue("fileId", "file")
	response := httptest.NewRecorder()

	controller.GetFileContents(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if model.requestedSnapshotID != "snapshot" || model.requestedFileID != "file" || model.requestedScope != reviewsnapshots.ScopeWorkingTree {
		t.Fatalf("FileContents() arguments = %q/%q/%q", model.requestedSnapshotID, model.requestedFileID, model.requestedScope)
	}
	var contents reviewsnapshots.FileContents
	if err := json.Unmarshal(response.Body.Bytes(), &contents); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if contents.ModifiedContent != "after" {
		t.Fatalf("contents = %#v", contents)
	}
}

func TestGetReviewSnapshotFileContentsMapsModelScopeError(t *testing.T) {
	model := &fakeReviewSnapshotsModel{fileContentsError: reviewsnapshots.InvalidScopeError{Scope: "invalid"}}
	controller := NewReviewSnapshots(model)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/review-snapshots/snapshot/files/file/contents?scope=invalid", nil)
	request.SetPathValue("snapshotId", "snapshot")
	request.SetPathValue("fileId", "file")
	response := httptest.NewRecorder()

	controller.GetFileContents(response, request)

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnprocessableEntity)
	}
	var problem Problem
	if err := json.Unmarshal(response.Body.Bytes(), &problem); err != nil {
		t.Fatalf("decoding problem: %v", err)
	}
	if problem.Code != "invalid_review_scope" {
		t.Fatalf("problem code = %q, want invalid_review_scope", problem.Code)
	}
}
