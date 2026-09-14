package reviewsnapshots

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// Model owns review snapshot creation, immutable resources, and file contents.
type Model struct {
	workspaces WorkspaceDiscovery
	git        Git
	github     GitHub
	files      FileSystem

	mu    sync.RWMutex
	items map[string]*snapshotRecord
}

// New constructs an application-scoped ReviewSnapshots Model.
func New(workspaces WorkspaceDiscovery, git Git, github GitHub, files FileSystem) *Model {
	return &Model{
		workspaces: workspaces,
		git:        git,
		github:     github,
		files:      files,
		items:      make(map[string]*snapshotRecord),
	}
}

// Create captures a detached point-in-time review snapshot for a workspace.
func (model *Model) Create(ctx context.Context, workspaceID string) (ReviewSnapshot, error) {
	workspacePath, found, err := model.workspaces.Resolve(workspaceID)
	if err != nil {
		return ReviewSnapshot{}, fmt.Errorf("resolving workspace %q: %w", workspaceID, err)
	}
	if !found {
		return ReviewSnapshot{}, WorkspaceNotFoundError{WorkspaceID: workspaceID}
	}

	window, err := model.buildWindowData(ctx, workspacePath)
	if err != nil {
		return ReviewSnapshot{}, err
	}
	pinWindowRevisions(ctx, &window, model.git)
	if err := captureWorkingTreeContents(model.files, &window); err != nil {
		return ReviewSnapshot{}, fmt.Errorf("capturing working tree contents: %w", err)
	}

	snapshotID, err := newSnapshotID()
	if err != nil {
		return ReviewSnapshot{}, err
	}

	var branch *SnapshotBranch
	if window.branchName != "" {
		branch = &SnapshotBranch{
			Ref:  "refs/heads/" + window.branchName,
			Name: window.branchName,
		}
	}

	var snapshotPullRequest *SnapshotPullRequest
	if window.pullRequest != nil {
		snapshotPullRequest = &SnapshotPullRequest{
			Number:  window.pullRequest.number,
			URL:     window.pullRequest.url,
			BaseRef: branchReference(window.pullRequest.baseRefName),
			HeadRef: branchReference(window.pullRequest.headRefName),
		}
	}

	snapshot := ReviewSnapshot{
		ID:          snapshotID,
		WorkspaceID: workspaceID,
		Branch:      branch,
		PullRequest: snapshotPullRequest,
		Files:       cloneReviewFiles(window.files),
		CreatedAt:   time.Now().UTC(),
	}

	model.mu.Lock()
	model.items[snapshotID] = &snapshotRecord{
		snapshot:              snapshot,
		window:                window,
		annotations:           make([]Annotation, 0),
		annotationSubscribers: make(map[uint64]chan AnnotationRevision),
	}
	model.mu.Unlock()

	return cloneReviewSnapshot(snapshot), nil
}

// List returns detached snapshots belonging to one configured workspace, newest first.
func (model *Model) List(ctx context.Context, workspaceID string) ([]ReviewSnapshot, error) {
	_, found, err := model.workspaces.Resolve(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("resolving workspace %q: %w", workspaceID, err)
	}
	if !found {
		return nil, WorkspaceNotFoundError{WorkspaceID: workspaceID}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	model.mu.RLock()
	snapshots := make([]ReviewSnapshot, 0)
	for _, record := range model.items {
		if record.snapshot.WorkspaceID == workspaceID {
			snapshots = append(snapshots, cloneReviewSnapshot(record.snapshot))
		}
	}
	model.mu.RUnlock()

	sort.Slice(snapshots, func(firstIndex int, secondIndex int) bool {
		if snapshots[firstIndex].CreatedAt.Equal(snapshots[secondIndex].CreatedAt) {
			return snapshots[firstIndex].ID < snapshots[secondIndex].ID
		}
		return snapshots[firstIndex].CreatedAt.After(snapshots[secondIndex].CreatedAt)
	})
	return snapshots, nil
}

// Get returns a detached copy of an in-memory snapshot.
func (model *Model) Get(snapshotID string) (ReviewSnapshot, error) {
	model.mu.RLock()
	record, found := model.items[snapshotID]
	model.mu.RUnlock()
	if !found {
		return ReviewSnapshot{}, SnapshotNotFoundError{SnapshotID: snapshotID}
	}

	return cloneReviewSnapshot(record.snapshot), nil
}

// FileContents returns contents for one snapshot-scoped file and comparison.
func (model *Model) FileContents(ctx context.Context, snapshotID string, fileID string, scope Scope) (FileContents, error) {
	if !isValidScope(scope) {
		return FileContents{}, InvalidScopeError{Scope: scope}
	}

	model.mu.RLock()
	record, found := model.items[snapshotID]
	model.mu.RUnlock()
	if !found {
		return FileContents{}, SnapshotNotFoundError{SnapshotID: snapshotID}
	}

	for _, file := range record.window.files {
		if file.ID == fileID {
			return model.loadFileContents(ctx, record.window.repoRoot, file, scope)
		}
	}

	return FileContents{}, SnapshotFileNotFoundError{SnapshotID: snapshotID, FileID: fileID}
}

// Delete removes one snapshot and closes its annotation subscriptions.
func (model *Model) Delete(snapshotID string) error {
	model.mu.Lock()
	defer model.mu.Unlock()

	record, found := model.items[snapshotID]
	if !found {
		return SnapshotNotFoundError{SnapshotID: snapshotID}
	}

	delete(model.items, snapshotID)
	for _, updates := range record.annotationSubscribers {
		close(updates)
	}
	return nil
}

// Close removes every snapshot and closes all annotation subscriptions.
func (model *Model) Close() {
	model.mu.Lock()
	defer model.mu.Unlock()

	for snapshotID, record := range model.items {
		delete(model.items, snapshotID)
		for _, updates := range record.annotationSubscribers {
			close(updates)
		}
	}
}

func cloneReviewSnapshot(snapshot ReviewSnapshot) ReviewSnapshot {
	cloned := snapshot
	cloned.Files = cloneReviewFiles(snapshot.Files)
	if snapshot.Branch != nil {
		branch := *snapshot.Branch
		branch.Remote = cloneString(snapshot.Branch.Remote)
		cloned.Branch = &branch
	}
	if snapshot.PullRequest != nil {
		pullRequest := *snapshot.PullRequest
		cloned.PullRequest = &pullRequest
	}
	return cloned
}

func cloneReviewFiles(files []File) []File {
	cloned := make([]File, len(files))
	for index, file := range files {
		cloned[index] = file
		cloned[index].WorktreeStatus = cloneChangeStatus(file.WorktreeStatus)
		cloned[index].GitDiff = cloneFileComparison(file.GitDiff)
		cloned[index].LastCommit = cloneFileComparison(file.LastCommit)
		cloned[index].PullRequest = cloneFileComparison(file.PullRequest)
	}
	return cloned
}

func cloneFileComparison(comparison *FileComparison) *FileComparison {
	if comparison == nil {
		return nil
	}

	cloned := *comparison
	cloned.OldPath = cloneString(comparison.OldPath)
	cloned.NewPath = cloneString(comparison.NewPath)
	cloned.originalRevision = ""
	cloned.modifiedRevision = ""
	cloned.capturedModifiedContent = nil
	return &cloned
}

func cloneChangeStatus(status *ChangeStatus) *ChangeStatus {
	if status == nil {
		return nil
	}

	cloned := *status
	return &cloned
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}

	cloned := *value
	return &cloned
}

func pinWindowRevisions(ctx context.Context, window *windowData, git Git) {
	headRevision := commitRevision(ctx, window.repoRoot, "HEAD", git)
	parentRevision := commitRevision(ctx, window.repoRoot, "HEAD^", git)
	for index := range window.files {
		file := &window.files[index]
		if file.GitDiff != nil {
			file.GitDiff.originalRevision = headRevision
		}
		if file.LastCommit != nil {
			file.LastCommit.originalRevision = parentRevision
			file.LastCommit.modifiedRevision = headRevision
		}
		if file.PullRequest != nil && file.PullRequest.modifiedRevision == "HEAD" {
			file.PullRequest.modifiedRevision = headRevision
		}
	}
}

func captureWorkingTreeContents(files FileSystem, window *windowData) error {
	for index := range window.files {
		comparison := window.files[index].GitDiff
		if comparison == nil || comparison.NewPath == nil {
			continue
		}

		content, err := workingTreeContent(files, window.repoRoot, *comparison.NewPath)
		if err != nil {
			return err
		}
		comparison.capturedModifiedContent = &content
	}
	return nil
}

func newSnapshotID() (string, error) {
	bytes := make([]byte, 12)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generating review snapshot ID: %w", err)
	}

	return "review_snapshot_" + hex.EncodeToString(bytes), nil
}

func branchReference(branchName string) string {
	if branchName == "" || strings.HasPrefix(branchName, "refs/") {
		return branchName
	}

	return "refs/heads/" + branchName
}
