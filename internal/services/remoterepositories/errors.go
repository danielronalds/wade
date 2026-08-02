package remoterepositories

import "fmt"

type InvalidRemoteRepositoryIDError struct {
	RemoteRepositoryID string
}

func (e InvalidRemoteRepositoryIDError) Error() string {
	return fmt.Sprintf("invalid remote repository ID %q", e.RemoteRepositoryID)
}

type WorkspaceDirectoryNotConfiguredError struct {
	WorkspaceDirectory string
}

func (e WorkspaceDirectoryNotConfiguredError) Error() string {
	return fmt.Sprintf("workspace directory %q is not configured", e.WorkspaceDirectory)
}

type WorkspaceAlreadyExistsError struct {
	WorkspaceID string
}

func (e WorkspaceAlreadyExistsError) Error() string {
	return fmt.Sprintf("workspace %q already exists", e.WorkspaceID)
}
