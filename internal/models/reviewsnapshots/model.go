package reviewsnapshots

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
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
	items map[string]snapshotRecord
}

// New constructs an application-scoped ReviewSnapshots Model.
func New(workspaces WorkspaceDiscovery, git Git, github GitHub, files FileSystem) *Model {
	return &Model{
		workspaces: workspaces,
		git:        git,
		github:     github,
		files:      files,
		items:      make(map[string]snapshotRecord),
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
	model.items[snapshotID] = snapshotRecord{
		snapshot: snapshot,
		window:   window,
	}
	model.mu.Unlock()

	return cloneReviewSnapshot(snapshot), nil
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

// Delete removes one snapshot from the in-memory registry.
func (model *Model) Delete(snapshotID string) error {
	model.mu.Lock()
	defer model.mu.Unlock()

	if _, found := model.items[snapshotID]; !found {
		return SnapshotNotFoundError{SnapshotID: snapshotID}
	}

	delete(model.items, snapshotID)
	return nil
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
