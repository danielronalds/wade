package repositories

import "path/filepath"

func validateWorkspaceID(workspaceID string) error {
	if workspaceID == "" || workspaceID == "." || workspaceID == ".." {
		return InvalidWorkspaceIDError{WorkspaceID: workspaceID}
	}
	if filepath.IsAbs(workspaceID) || filepath.Base(workspaceID) != workspaceID {
		return InvalidWorkspaceIDError{WorkspaceID: workspaceID}
	}

	return nil
}

func validateRepositoryID(repositoryID string) error {
	if repositoryID == "" || repositoryID == "." || repositoryID == ".." {
		return InvalidRepositoryIDError{RepositoryID: repositoryID}
	}
	if filepath.IsAbs(repositoryID) || filepath.Base(repositoryID) != repositoryID {
		return InvalidRepositoryIDError{RepositoryID: repositoryID}
	}

	return nil
}
