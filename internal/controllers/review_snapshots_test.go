package controllers

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

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

func TestListWorkspaceReviewSnapshotsReturnsCollection(t *testing.T) {
	model := &fakeReviewSnapshotsModel{snapshots: []reviewsnapshots.ReviewSnapshot{{ID: "snapshot-1", WorkspaceID: "wade"}}}
	controller := NewReviewSnapshots(model)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/wade/review-snapshots", nil)
	request.SetPathValue("workspaceId", "wade")
	response := httptest.NewRecorder()

	controller.List(response, request)

	if response.Code != http.StatusOK || model.listedWorkspaceID != "wade" {
		t.Fatalf("List() response/model = %d/%q", response.Code, model.listedWorkspaceID)
	}
	var collection ReviewSnapshotList
	if err := json.Unmarshal(response.Body.Bytes(), &collection); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if len(collection.Items) != 1 || collection.Items[0].ID != "snapshot-1" {
		t.Fatalf("collection = %#v", collection)
	}
}

func TestCreateReviewAnnotationReturnsResourceAndExactLocation(t *testing.T) {
	model := &fakeReviewSnapshotsModel{annotation: reviewsnapshots.Annotation{
		ID:         "annotation/id",
		SnapshotID: "snapshot/id",
		FileID:     "file/id",
	}}
	controller := NewReviewSnapshots(model)
	body := `{"fileId":"file/id","scope":"working-tree","side":"modified","startLine":2,"endLine":4,"summary":"Why this exists"}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/review-snapshots/snapshot%2Fid/annotations", strings.NewReader(body))
	request.SetPathValue("snapshotId", "snapshot/id")
	response := httptest.NewRecorder()

	controller.CreateAnnotation(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusCreated, response.Body.String())
	}
	if location := response.Header().Get("Location"); location != "/api/v1/review-snapshots/snapshot%2Fid/annotations/annotation%2Fid" {
		t.Fatalf("Location = %q", location)
	}
	if model.annotationRequest.FileID != "file/id" || model.annotationRequest.StartLine != 2 || model.annotationRequest.EndLine != 4 {
		t.Fatalf("CreateAnnotation() request = %#v", model.annotationRequest)
	}
}

func TestCreateReviewAnnotationRejectsMalformedAndIncompleteBodiesWithoutModelMutation(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "invalid JSON", body: `{`},
		{name: "unknown field", body: `{"fileId":"file","scope":"working-tree","side":"modified","startLine":1,"endLine":1,"summary":"note","extra":true}`},
		{name: "missing required field", body: `{"fileId":"file","scope":"working-tree","side":"modified","startLine":1,"summary":"note"}`},
		{name: "multiple values", body: `{} {}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			model := &fakeReviewSnapshotsModel{}
			controller := NewReviewSnapshots(model)
			request := httptest.NewRequest(http.MethodPost, "/annotations", strings.NewReader(test.body))
			request.SetPathValue("snapshotId", "snapshot")
			response := httptest.NewRecorder()

			controller.CreateAnnotation(response, request)

			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
			}
			if model.requestedSnapshotID != "" {
				t.Fatalf("CreateAnnotation() called for %q", model.requestedSnapshotID)
			}
		})
	}
}

func TestReviewAnnotationCollectionAndDeleteUseSnapshotAndAnnotationIDs(t *testing.T) {
	model := &fakeReviewSnapshotsModel{annotations: []reviewsnapshots.Annotation{{ID: "annotation-1"}}}
	controller := NewReviewSnapshots(model)
	listRequest := httptest.NewRequest(http.MethodGet, "/annotations", nil)
	listRequest.SetPathValue("snapshotId", "snapshot-1")
	listResponse := httptest.NewRecorder()

	controller.ListAnnotations(listResponse, listRequest)

	var collection AnnotationList
	if err := json.Unmarshal(listResponse.Body.Bytes(), &collection); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if listResponse.Code != http.StatusOK || len(collection.Items) != 1 {
		t.Fatalf("list response = %d/%#v", listResponse.Code, collection)
	}

	deleteRequest := httptest.NewRequest(http.MethodDelete, "/annotations/annotation-1", nil)
	deleteRequest.SetPathValue("snapshotId", "snapshot-1")
	deleteRequest.SetPathValue("annotationId", "annotation-1")
	deleteResponse := httptest.NewRecorder()
	controller.DeleteAnnotation(deleteResponse, deleteRequest)

	if deleteResponse.Code != http.StatusNoContent || model.requestedSnapshotID != "snapshot-1" || model.deletedAnnotationID != "annotation-1" {
		t.Fatalf("delete response/model = %d/%q/%q", deleteResponse.Code, model.requestedSnapshotID, model.deletedAnnotationID)
	}
}

func TestReviewAnnotationEventsStreamsRevisionAndClosesSubscriptionOnCancellation(t *testing.T) {
	updates := make(chan reviewsnapshots.AnnotationRevision, 1)
	updates <- reviewsnapshots.AnnotationRevision{Revision: 7}
	subscription := &annotationSubscriptionStub{updates: updates, closed: make(chan struct{})}
	model := &fakeReviewSnapshotsModel{subscription: subscription}
	controller := NewReviewSnapshots(model)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/review-snapshots/{snapshotId}/annotations/events", controller.AnnotationEvents)
	server := httptest.NewServer(mux)
	defer server.Close()

	response, err := http.Get(server.URL + "/api/v1/review-snapshots/snapshot-1/annotations/events")
	if err != nil {
		t.Fatalf("GET event stream: %v", err)
	}
	if response.StatusCode != http.StatusOK || response.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatalf("event response = %d/%q", response.StatusCode, response.Header.Get("Content-Type"))
	}

	reader := bufio.NewReader(response.Body)
	var event strings.Builder
	for {
		line, readError := reader.ReadString('\n')
		if readError != nil {
			t.Fatalf("reading event stream: %v", readError)
		}
		event.WriteString(line)
		if line == "\n" {
			break
		}
	}
	for _, expected := range []string{"event: annotations-changed", `data: {"revision":7}`} {
		if !strings.Contains(event.String(), expected) {
			t.Fatalf("event %q does not contain %q", event.String(), expected)
		}
	}

	if err := response.Body.Close(); err != nil {
		t.Fatalf("closing event response: %v", err)
	}
	select {
	case <-subscription.closed:
	case <-time.After(time.Second):
		t.Fatal("subscription was not closed after request cancellation")
	}
}

type annotationSubscriptionStub struct {
	updates <-chan reviewsnapshots.AnnotationRevision
	closed  chan struct{}
	once    sync.Once
}

func (subscription *annotationSubscriptionStub) Updates() <-chan reviewsnapshots.AnnotationRevision {
	return subscription.updates
}

func (subscription *annotationSubscriptionStub) Close() {
	subscription.once.Do(func() { close(subscription.closed) })
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
