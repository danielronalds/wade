package workspaces

import "fmt"

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
