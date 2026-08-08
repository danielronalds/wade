package repositories

import "fmt"

type InvalidRepositoryIDError struct {
	RepositoryID string
}

func (e InvalidRepositoryIDError) Error() string {
	return fmt.Sprintf("invalid repository ID %q", e.RepositoryID)
}

type RepositoryNotFoundError struct {
	RepositoryID string
}

func (e RepositoryNotFoundError) Error() string {
	return fmt.Sprintf("repository %q not found", e.RepositoryID)
}

type RepositoryIDConflictError struct {
	RepositoryID string
}

func (e RepositoryIDConflictError) Error() string {
	return fmt.Sprintf("repository ID %q identifies more than one local repository", e.RepositoryID)
}

type InvalidWorkspaceIDError struct {
	WorkspaceID string
}

func (e InvalidWorkspaceIDError) Error() string {
	return fmt.Sprintf("invalid workspace ID %q", e.WorkspaceID)
}

type WorkspaceNotFoundError struct {
	WorkspaceID string
}

func (e WorkspaceNotFoundError) Error() string {
	return fmt.Sprintf("workspace %q not found", e.WorkspaceID)
}
