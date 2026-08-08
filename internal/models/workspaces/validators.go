package workspaces

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
