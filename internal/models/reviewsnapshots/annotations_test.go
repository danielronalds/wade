package reviewsnapshots

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestListSnapshotsReturnsWorkspaceSnapshotsNewestFirstAndDetached(t *testing.T) {
	model, _ := newSnapshotTestModel()
	first, err := model.Create(context.Background(), "wade")
	if err != nil {
		t.Fatalf("Create(first) error = %v, want nil", err)
	}
	time.Sleep(time.Millisecond)
	second, err := model.Create(context.Background(), "wade")
	if err != nil {
		t.Fatalf("Create(second) error = %v, want nil", err)
	}
	if _, err := model.Create(context.Background(), "other"); err != nil {
		t.Fatalf("Create(other) error = %v, want nil", err)
	}

	listed, err := model.List(context.Background(), "wade")
	if err != nil {
		t.Fatalf("List() error = %v, want nil", err)
	}
	if len(listed) != 2 || listed[0].ID != second.ID || listed[1].ID != first.ID {
		t.Fatalf("List() = %#v, want second then first", listed)
	}
	listed[0].Files[0].ID = "mutated"
	reloaded, err := model.List(context.Background(), "wade")
	if err != nil || reloaded[0].Files[0].ID == "mutated" {
		t.Fatalf("List() returned mutable snapshots: %#v, %v", reloaded, err)
	}

	empty, err := model.List(context.Background(), "unused")
	if err != nil || empty == nil || len(empty) != 0 {
		t.Fatalf("List(unused) = %#v, %v, want empty collection", empty, err)
	}

	missingModel := New(workspaceDiscoveryStub{}, gitStub{}, gitHubStub{}, &fileSystemStub{})
	_, err = missingModel.List(context.Background(), "missing")
	var workspaceNotFound WorkspaceNotFoundError
	if !errors.As(err, &workspaceNotFound) {
		t.Fatalf("List(missing) error = %v, want WorkspaceNotFoundError", err)
	}
}

func TestAnnotationLifecyclePublishesDetachedCollectionRevisions(t *testing.T) {
	model, _ := newSnapshotTestModel()
	snapshot, err := model.Create(context.Background(), "wade")
	if err != nil {
		t.Fatalf("Create() error = %v, want nil", err)
	}
	file := findReviewFile(t, snapshot.Files, "tracked.txt")

	subscription, err := model.SubscribeAnnotations(snapshot.ID)
	if err != nil {
		t.Fatalf("SubscribeAnnotations() error = %v, want nil", err)
	}
	defer subscription.Close()
	assertAnnotationRevision(t, subscription.Updates(), 0)

	rationale := "  Retains the snapshot invariant.  "
	author := "  Pi  "
	annotation, err := model.CreateAnnotation(context.Background(), snapshot.ID, CreateAnnotationRequest{
		FileID:    file.ID,
		Scope:     ScopeWorkingTree,
		Side:      AnnotationSideModified,
		StartLine: 1,
		EndLine:   1,
		Summary:   "  Validate before commit.  ",
		Rationale: &rationale,
		Author:    &author,
	})
	if err != nil {
		t.Fatalf("CreateAnnotation() error = %v, want nil", err)
	}
	if annotation.ID == "" || annotation.SnapshotID != snapshot.ID || annotation.Summary != "Validate before commit." {
		t.Fatalf("CreateAnnotation() = %#v", annotation)
	}
	if annotation.Rationale == nil || *annotation.Rationale != "Retains the snapshot invariant." || annotation.Author == nil || *annotation.Author != "Pi" {
		t.Fatalf("CreateAnnotation() optional text = %#v/%#v", annotation.Rationale, annotation.Author)
	}
	assertAnnotationRevision(t, subscription.Updates(), 1)

	*annotation.Rationale = "mutated"
	listed, err := model.ListAnnotations(snapshot.ID)
	if err != nil {
		t.Fatalf("ListAnnotations() error = %v, want nil", err)
	}
	if len(listed) != 1 || listed[0].Rationale == nil || *listed[0].Rationale != "Retains the snapshot invariant." {
		t.Fatalf("ListAnnotations() = %#v, want detached annotation", listed)
	}
	*listed[0].Author = "mutated"
	reloaded, err := model.ListAnnotations(snapshot.ID)
	if err != nil || *reloaded[0].Author != "Pi" {
		t.Fatalf("ListAnnotations() after mutation = %#v, %v", reloaded, err)
	}

	if err := model.DeleteAnnotation(snapshot.ID, annotation.ID); err != nil {
		t.Fatalf("DeleteAnnotation() error = %v, want nil", err)
	}
	assertAnnotationRevision(t, subscription.Updates(), 2)
	listed, err = model.ListAnnotations(snapshot.ID)
	if err != nil || len(listed) != 0 || listed == nil {
		t.Fatalf("ListAnnotations() after delete = %#v, %v, want empty collection", listed, err)
	}

	err = model.DeleteAnnotation(snapshot.ID, annotation.ID)
	var annotationNotFound AnnotationNotFoundError
	if !errors.As(err, &annotationNotFound) {
		t.Fatalf("DeleteAnnotation() error = %v, want AnnotationNotFoundError", err)
	}
	assertNoAnnotationRevision(t, subscription.Updates())

	if err := model.Delete(snapshot.ID); err != nil {
		t.Fatalf("Delete() error = %v, want nil", err)
	}
	if _, open := <-subscription.Updates(); open {
		t.Fatal("subscription remained open after snapshot deletion")
	}
	_, err = model.ListAnnotations(snapshot.ID)
	var snapshotNotFound SnapshotNotFoundError
	if !errors.As(err, &snapshotNotFound) {
		t.Fatalf("ListAnnotations() error = %v, want SnapshotNotFoundError", err)
	}
}

func TestCreateAnnotationValidatesCapturedScopeSideRangeAndText(t *testing.T) {
	model, _ := newSnapshotTestModel()
	snapshot, err := model.Create(context.Background(), "wade")
	if err != nil {
		t.Fatalf("Create() error = %v, want nil", err)
	}
	tracked := findReviewFile(t, snapshot.Files, "tracked.txt")
	added := findReviewFile(t, snapshot.Files, "untracked.txt")

	tests := []struct {
		name    string
		request CreateAnnotationRequest
		detail  string
	}{
		{
			name:    "current scope",
			request: validAnnotationRequest(tracked.ID, ScopeCurrent, AnnotationSideModified),
			detail:  "scope",
		},
		{
			name:    "file outside scope",
			request: validAnnotationRequest(added.ID, ScopeLastCommit, AnnotationSideModified),
			detail:  "fileId",
		},
		{
			name:    "missing added original side",
			request: validAnnotationRequest(added.ID, ScopeWorkingTree, AnnotationSideOriginal),
			detail:  "original side",
		},
		{
			name: "zero start line",
			request: func() CreateAnnotationRequest {
				request := validAnnotationRequest(tracked.ID, ScopeWorkingTree, AnnotationSideModified)
				request.StartLine = 0
				return request
			}(),
			detail: "startLine",
		},
		{
			name: "reversed range",
			request: func() CreateAnnotationRequest {
				request := validAnnotationRequest(tracked.ID, ScopeWorkingTree, AnnotationSideModified)
				request.StartLine = 2
				return request
			}(),
			detail: "endLine",
		},
		{
			name: "range beyond captured content",
			request: func() CreateAnnotationRequest {
				request := validAnnotationRequest(tracked.ID, ScopeWorkingTree, AnnotationSideModified)
				request.EndLine = 2
				return request
			}(),
			detail: "captured lines",
		},
		{
			name: "empty summary",
			request: func() CreateAnnotationRequest {
				request := validAnnotationRequest(tracked.ID, ScopeWorkingTree, AnnotationSideModified)
				request.Summary = " \n "
				return request
			}(),
			detail: "summary",
		},
		{
			name: "oversized summary",
			request: func() CreateAnnotationRequest {
				request := validAnnotationRequest(tracked.ID, ScopeWorkingTree, AnnotationSideModified)
				request.Summary = strings.Repeat("x", maximumAnnotationSummaryBytes+1)
				return request
			}(),
			detail: "summary",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := model.CreateAnnotation(context.Background(), snapshot.ID, test.request)
			var invalidAnnotation InvalidAnnotationError
			if !errors.As(err, &invalidAnnotation) || !strings.Contains(err.Error(), test.detail) {
				t.Fatalf("CreateAnnotation() error = %v, want InvalidAnnotationError containing %q", err, test.detail)
			}
		})
	}

	listed, err := model.ListAnnotations(snapshot.ID)
	if err != nil || len(listed) != 0 {
		t.Fatalf("ListAnnotations() = %#v, %v, want no mutation", listed, err)
	}
}

func TestAnnotationLimitIsEnforcedDuringConcurrentCreation(t *testing.T) {
	model, _ := newSnapshotTestModel()
	snapshot, err := model.Create(context.Background(), "wade")
	if err != nil {
		t.Fatalf("Create() error = %v, want nil", err)
	}
	file := findReviewFile(t, snapshot.Files, "tracked.txt")
	request := validAnnotationRequest(file.ID, ScopeWorkingTree, AnnotationSideModified)

	var successes atomic.Int64
	var wait sync.WaitGroup
	for range maximumAnnotationsPerSnapshot + 20 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			if _, err := model.CreateAnnotation(context.Background(), snapshot.ID, request); err == nil {
				successes.Add(1)
				return
			} else {
				var invalidAnnotation InvalidAnnotationError
				if !errors.As(err, &invalidAnnotation) {
					t.Errorf("CreateAnnotation() error = %v, want InvalidAnnotationError", err)
				}
			}
		}()
	}
	wait.Wait()

	if successes.Load() != maximumAnnotationsPerSnapshot {
		t.Fatalf("successful creates = %d, want %d", successes.Load(), maximumAnnotationsPerSnapshot)
	}
	listed, err := model.ListAnnotations(snapshot.ID)
	if err != nil || len(listed) != maximumAnnotationsPerSnapshot {
		t.Fatalf("ListAnnotations() count = %d, %v", len(listed), err)
	}
}

func TestSnapshotDeletionRacingAnnotationValidationCannotRetainAnnotation(t *testing.T) {
	model, _ := newSnapshotTestModel()
	snapshot, err := model.Create(context.Background(), "wade")
	if err != nil {
		t.Fatalf("Create() error = %v, want nil", err)
	}
	file := findReviewFile(t, snapshot.Files, "tracked.txt")

	baseGit := model.git.(gitStub)
	blockingGit := &blockingRevisionGit{
		gitStub: baseGit,
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	model.git = blockingGit

	result := make(chan error, 1)
	go func() {
		_, createError := model.CreateAnnotation(
			context.Background(),
			snapshot.ID,
			validAnnotationRequest(file.ID, ScopeWorkingTree, AnnotationSideOriginal),
		)
		result <- createError
	}()

	<-blockingGit.started
	if err := model.Delete(snapshot.ID); err != nil {
		t.Fatalf("Delete() error = %v, want nil", err)
	}
	close(blockingGit.release)

	err = <-result
	var snapshotNotFound SnapshotNotFoundError
	if !errors.As(err, &snapshotNotFound) {
		t.Fatalf("CreateAnnotation() error = %v, want SnapshotNotFoundError", err)
	}
}

func TestSlowAnnotationSubscriberReceivesLatestCoalescedRevision(t *testing.T) {
	model, _ := newSnapshotTestModel()
	snapshot, err := model.Create(context.Background(), "wade")
	if err != nil {
		t.Fatalf("Create() error = %v, want nil", err)
	}
	file := findReviewFile(t, snapshot.Files, "tracked.txt")
	subscription, err := model.SubscribeAnnotations(snapshot.ID)
	if err != nil {
		t.Fatalf("SubscribeAnnotations() error = %v, want nil", err)
	}
	defer subscription.Close()

	request := validAnnotationRequest(file.ID, ScopeWorkingTree, AnnotationSideModified)
	for range 5 {
		if _, err := model.CreateAnnotation(context.Background(), snapshot.ID, request); err != nil {
			t.Fatalf("CreateAnnotation() error = %v, want nil", err)
		}
	}

	assertAnnotationRevision(t, subscription.Updates(), 5)
}

type blockingRevisionGit struct {
	gitStub
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func (stub *blockingRevisionGit) RevisionContent(_ context.Context, _ string, revision string, filePath string) ([]byte, error) {
	stub.once.Do(func() { close(stub.started) })
	<-stub.release
	return stub.gitStub.RevisionContent(context.Background(), "", revision, filePath)
}

func validAnnotationRequest(fileID string, scope Scope, side AnnotationSide) CreateAnnotationRequest {
	return CreateAnnotationRequest{
		FileID:    fileID,
		Scope:     scope,
		Side:      side,
		StartLine: 1,
		EndLine:   1,
		Summary:   "Explain this line",
	}
}

func assertAnnotationRevision(t *testing.T, updates <-chan AnnotationRevision, want uint64) {
	t.Helper()
	select {
	case update, open := <-updates:
		if !open || update.Revision != want {
			t.Fatalf("annotation revision = %#v/open:%t, want %d", update, open, want)
		}
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for annotation revision %d", want)
	}
}

func assertNoAnnotationRevision(t *testing.T, updates <-chan AnnotationRevision) {
	t.Helper()
	select {
	case update := <-updates:
		t.Fatalf("unexpected annotation revision %#v", update)
	case <-time.After(20 * time.Millisecond):
	}
}
