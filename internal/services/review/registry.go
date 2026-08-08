package review

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

func NewService(
	workspaces WorkspaceRepository,
	git gitRepository,
	github gitHubRepository,
	files fileRepository,
) *Service {
	return &Service{
		workspaces: workspaces,
		git:        git,
		github:     github,
		files:      files,
		state: &snapshotState{
			items: make(map[string]snapshotRecord),
		},
	}
}

func (s *Service) CreateSnapshot(ctx context.Context, workspaceID string) (ReviewSnapshot, error) {
	workspacePath, found, err := s.workspaces.Resolve(workspaceID)
	if err != nil {
		return ReviewSnapshot{}, err
	}
	if !found {
		return ReviewSnapshot{}, WorkspaceNotFoundError{WorkspaceID: workspaceID}
	}

	window, err := s.BuildWindowData(ctx, workspacePath)
	if err != nil {
		return ReviewSnapshot{}, err
	}
	pinWindowRevisions(ctx, &window, s.git)

	snapshotID, err := newSnapshotID()
	if err != nil {
		return ReviewSnapshot{}, err
	}

	var branch *SnapshotBranch
	if window.BranchName != "" {
		branch = &SnapshotBranch{
			Ref:  "refs/heads/" + window.BranchName,
			Name: window.BranchName,
		}
	}

	var pullRequest *SnapshotPullRequest
	if window.PullRequest != nil {
		pullRequest = &SnapshotPullRequest{
			Number:  window.PullRequest.Number,
			URL:     window.PullRequest.URL,
			BaseRef: branchReference(window.PullRequest.BaseRefName),
			HeadRef: branchReference(window.PullRequest.HeadRefName),
		}
	}

	snapshot := ReviewSnapshot{
		ID:          snapshotID,
		WorkspaceID: workspaceID,
		Branch:      branch,
		PullRequest: pullRequest,
		Files:       cloneReviewFiles(window.Files),
		CreatedAt:   time.Now().UTC(),
	}

	s.state.mu.Lock()
	s.state.items[snapshotID] = snapshotRecord{
		snapshot:      snapshot,
		workspacePath: workspacePath,
		window:        window,
	}
	s.state.mu.Unlock()

	return cloneReviewSnapshot(snapshot), nil
}

func (s *Service) GetSnapshot(snapshotID string) (ReviewSnapshot, error) {
	s.state.mu.RLock()
	record, found := s.state.items[snapshotID]
	s.state.mu.RUnlock()
	if !found {
		return ReviewSnapshot{}, SnapshotNotFoundError{SnapshotID: snapshotID}
	}

	return cloneReviewSnapshot(record.snapshot), nil
}

func (s *Service) LoadSnapshotFileContents(
	ctx context.Context,
	snapshotID string,
	fileID string,
	scope Scope,
) (FileContents, error) {
	if !IsValidScope(scope) {
		return FileContents{}, InvalidScopeError{Scope: scope}
	}

	s.state.mu.RLock()
	record, found := s.state.items[snapshotID]
	s.state.mu.RUnlock()
	if !found {
		return FileContents{}, SnapshotNotFoundError{SnapshotID: snapshotID}
	}

	for _, file := range record.window.Files {
		if file.ID == fileID {
			return s.LoadFileContents(ctx, record.window.RepoRoot, file, scope)
		}
	}

	return FileContents{}, SnapshotFileNotFoundError{SnapshotID: snapshotID, FileID: fileID}
}

func (s *Service) DeleteSnapshot(snapshotID string) error {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()

	if _, found := s.state.items[snapshotID]; !found {
		return SnapshotNotFoundError{SnapshotID: snapshotID}
	}

	delete(s.state.items, snapshotID)
	return nil
}

func cloneReviewSnapshot(snapshot ReviewSnapshot) ReviewSnapshot {
	cloned := snapshot
	cloned.Files = cloneReviewFiles(snapshot.Files)
	if snapshot.Branch != nil {
		branch := *snapshot.Branch
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

func pinWindowRevisions(ctx context.Context, window *WindowData, git gitRepository) {
	headRevision := commitRevision(ctx, window.RepoRoot, "HEAD", git)
	parentRevision := commitRevision(ctx, window.RepoRoot, "HEAD^", git)
	for index := range window.Files {
		file := &window.Files[index]
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
