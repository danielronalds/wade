package workspaces

import "fmt"

// InvalidRemoteRepositoryIDError reports a malformed provider repository identity.
type InvalidRemoteRepositoryIDError struct {
	RemoteRepositoryID string
}

func (e InvalidRemoteRepositoryIDError) Error() string {
	return fmt.Sprintf("invalid remote repository ID %q", e.RemoteRepositoryID)
}

// WorkspaceDirectoryNotConfiguredError reports an unconfigured clone destination.
type WorkspaceDirectoryNotConfiguredError struct {
	WorkspaceDirectory string
}

func (e WorkspaceDirectoryNotConfiguredError) Error() string {
	return fmt.Sprintf("workspace directory %q is not configured", e.WorkspaceDirectory)
}

// WorkspaceAlreadyExistsError reports a conflicting workspace identity or path.
type WorkspaceAlreadyExistsError struct {
	WorkspaceID string
}

func (e WorkspaceAlreadyExistsError) Error() string {
	return fmt.Sprintf("workspace %q already exists", e.WorkspaceID)
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
