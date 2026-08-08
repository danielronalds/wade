package reviewsnapshots

import "fmt"

// WorkspaceNotFoundError reports an unknown workspace identity.
type WorkspaceNotFoundError struct {
	WorkspaceID string
}

func (err WorkspaceNotFoundError) Error() string {
	return fmt.Sprintf("workspace %q not found", err.WorkspaceID)
}

// WorkspaceNotGitRepositoryError reports a workspace without a Git repository.
type WorkspaceNotGitRepositoryError struct{}

func (WorkspaceNotGitRepositoryError) Error() string {
	return "workspace is not a Git repository"
}

// InvalidScopeError reports an unsupported review comparison scope.
type InvalidScopeError struct {
	Scope Scope
}

func (err InvalidScopeError) Error() string {
	return fmt.Sprintf("invalid review scope %q", err.Scope)
}

// SnapshotNotFoundError reports an unknown or deleted snapshot.
type SnapshotNotFoundError struct {
	SnapshotID string
}

func (err SnapshotNotFoundError) Error() string {
	return fmt.Sprintf("review snapshot %q not found", err.SnapshotID)
}

// SnapshotFileNotFoundError reports an unknown file identity within a snapshot.
type SnapshotFileNotFoundError struct {
	SnapshotID string
	FileID     string
}

func (err SnapshotFileNotFoundError) Error() string {
	return fmt.Sprintf("file %q was not found in review snapshot %q", err.FileID, err.SnapshotID)
}
