package repositories

import "fmt"

// InvalidRepositoryIDError reports a malformed repository identity.
type InvalidRepositoryIDError struct {
	RepositoryID string
}

func (e InvalidRepositoryIDError) Error() string {
	return fmt.Sprintf("invalid repository ID %q", e.RepositoryID)
}

// RepositoryNotFoundError reports an unknown repository identity.
type RepositoryNotFoundError struct {
	RepositoryID string
}

func (e RepositoryNotFoundError) Error() string {
	return fmt.Sprintf("repository %q not found", e.RepositoryID)
}

// RepositoryIDConflictError reports an ambiguous local repository identity.
type RepositoryIDConflictError struct {
	RepositoryID string
}

func (e RepositoryIDConflictError) Error() string {
	return fmt.Sprintf("repository ID %q identifies more than one local repository", e.RepositoryID)
}

// InvalidWorkspaceIDError reports a malformed workspace identity.
type InvalidWorkspaceIDError struct {
	WorkspaceID string
}

func (e InvalidWorkspaceIDError) Error() string {
	return fmt.Sprintf("invalid workspace ID %q", e.WorkspaceID)
}

// WorkspaceNotFoundError reports an unknown workspace identity.
type WorkspaceNotFoundError struct {
	WorkspaceID string
}

func (e WorkspaceNotFoundError) Error() string {
	return fmt.Sprintf("workspace %q not found", e.WorkspaceID)
}
