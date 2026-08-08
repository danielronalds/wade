package controllers

import (
	"context"

	"wade/internal/models/remoterepositories"
	"wade/internal/models/repositories"
	"wade/internal/models/terminals"
	"wade/internal/models/workspaces"
)

// WorkspacesModel is the complete Workspaces surface consumed by controllers.
type WorkspacesModel interface {
	List(ctx context.Context) ([]workspaces.WorkspaceSummary, error)
	ListByIDs(ctx context.Context, workspaceIDs []string) ([]workspaces.WorkspaceSummary, error)
	Get(ctx context.Context, workspaceID string) (workspaces.Workspace, error)
	Materialise(ctx context.Context, request workspaces.MaterialiseRequest) (workspaces.Workspace, error)
	ResolveLinks(ctx context.Context, linkContext workspaces.LinkContext) (workspaces.WorkspaceLinks, error)
}

// RepositoriesModel is the complete Repositories surface consumed by controllers.
type RepositoriesModel interface {
	Get(ctx context.Context, repositoryID string) (repositories.Repository, error)
	ListWorkspaceContexts(ctx context.Context) ([]repositories.WorkspaceContext, error)
	ListWorkspaceContextsByIDs(ctx context.Context, workspaceIDs []string) (map[string]repositories.WorkspaceContext, error)
	GetWorkspaceContext(ctx context.Context, workspaceID string) (*repositories.WorkspaceContext, error)
	WorkspaceIDsByRemoteRepository(ctx context.Context, remoteRepositoryIDs []string) (map[string][]string, error)
	ListWorktrees(ctx context.Context, repositoryID string) ([]repositories.Worktree, error)
	GetWorktree(ctx context.Context, repositoryID string, worktreeID string) (repositories.Worktree, error)
	CreateWorktree(ctx context.Context, repositoryID string, request repositories.CreateWorktreeRequest) (repositories.Worktree, error)
	RemoveWorktree(ctx context.Context, repositoryID string, worktreeID string) (repositories.Worktree, error)
	ListBranches(ctx context.Context, repositoryID string, kind repositories.BranchKind) ([]repositories.Branch, error)
}

// RemoteRepositoriesModel is the complete RemoteRepositories surface consumed by controllers.
type RemoteRepositoriesModel interface {
	List(ctx context.Context) ([]remoterepositories.RemoteRepository, error)
}

// TerminalsModel is the complete Terminals surface consumed by controllers.
type TerminalsModel interface {
	List(ctx context.Context, workspaceID string) ([]terminals.Terminal, error)
	Get(ctx context.Context, workspaceID string, terminalID string) (terminals.Terminal, error)
	Put(ctx context.Context, workspaceID string, terminalID string) (terminals.Terminal, bool, error)
	Delete(ctx context.Context, workspaceID string, terminalID string) error
	DeleteAll(ctx context.Context, workspaceID string) (int, error)
	Input(ctx context.Context, input terminals.Input) error
	Connect(ctx context.Context, workspaceID string, terminalID string) (*terminals.TerminalSession, error)
	ActiveTerminalCount(workspaceID string) int
	ActiveWorkspaceIDs() []string
}
