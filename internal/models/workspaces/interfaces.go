package workspaces

import (
	"context"

	"wade/internal/infrastructure/linear"
)

// FileSystem provides workspace materialisation filesystem operations.
type FileSystem interface {
	EnsureDirectory(path string) error
	EnsurePathDoesNotExist(path string) error
}

// WorkspaceDiscovery provides fresh configured-workspace discovery.
type WorkspaceDiscovery interface {
	IDs() ([]string, error)
	Resolve(workspaceID string) (string, bool, error)
	Directories() []string
	Reload(directories []string)
}

// GitHub provides clone and optional pull request operations.
type GitHub interface {
	CloneRepository(ctx context.Context, nameWithOwner string, targetPath string) error
	PullRequestURL(ctx context.Context, repository string, branch string) (string, error)
}

// Linear resolves an optional ticket associated with a branch.
type Linear interface {
	TicketForBranch(workspace string, branch string) (*linear.Ticket, error)
}
