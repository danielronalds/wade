package review

import "fmt"

type WorkspaceNotGitRepositoryError struct{}

func (WorkspaceNotGitRepositoryError) Error() string {
	return "workspace is not a Git repository"
}

type InvalidScopeError struct {
	Scope Scope
}

func (e InvalidScopeError) Error() string {
	return fmt.Sprintf("invalid review scope %q", e.Scope)
}

type SnapshotNotFoundError struct {
	SnapshotID string
}

func (e SnapshotNotFoundError) Error() string {
	return fmt.Sprintf("review snapshot %q not found", e.SnapshotID)
}

type SnapshotFileNotFoundError struct {
	SnapshotID string
	FileID     string
}

func (e SnapshotFileNotFoundError) Error() string {
	return fmt.Sprintf("file %q was not found in review snapshot %q", e.FileID, e.SnapshotID)
}
