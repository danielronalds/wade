package repositories

import (
	"context"

	"wade/internal/infrastructure/git"
)

// WorkspaceDiscovery provides fresh configured-workspace discovery.
type WorkspaceDiscovery interface {
	IDs() ([]string, error)
	Resolve(workspaceID string) (string, bool, error)
	CanonicalPath(workspaceID string) (string, bool, error)
	IDForDirectory(directory string) (string, bool, error)
	IDsForDirectories(directories []string) ([]string, error)
}

// Git provides the cohesive Git operations used by the Repositories aggregate.
type Git interface {
	IsGitWorktree(ctx context.Context, workspacePath string) (bool, error)
	WorktreePaths(ctx context.Context, workspacePath string) ([]string, error)
	CommonDirectory(ctx context.Context, workspacePath string) (string, error)
	HeadReference(ctx context.Context, workspacePath string) (string, bool, error)
	HeadCommit(ctx context.Context, workspacePath string) (string, bool, error)
	OriginRemoteURL(ctx context.Context, workspacePath string) (string, bool, error)
	Worktrees(ctx context.Context, repositoryPath string) ([]git.Worktree, error)
	Remotes(ctx context.Context, repositoryPath string) ([]string, error)
	FetchRemote(ctx context.Context, repositoryPath string, remote string) error
	RemoteBranches(ctx context.Context, repositoryPath string) ([]string, error)
	LocalBranches(ctx context.Context, repositoryPath string) ([]string, error)
	ValidateBranchName(ctx context.Context, repositoryPath string, branch string) error
	AddWorktree(ctx context.Context, repositoryPath string, targetPath string, branch string) error
	AddTrackingWorktree(ctx context.Context, repositoryPath string, localBranch string, targetPath string, remoteBranch string) error
	AddNewBranchWorktree(ctx context.Context, repositoryPath string, localBranch string, targetPath string) error
	RemoveWorktree(ctx context.Context, repositoryPath string, targetPath string) error
	PruneWorktrees(ctx context.Context, repositoryPath string) error
	DeleteBranch(ctx context.Context, repositoryPath string, branch string) error
	IgnoredPaths(ctx context.Context, repositoryPath string) ([]string, error)
}

// FileSystem provides file copying for worktree creation.
type FileSystem interface {
	CopyPath(source string, destination string) error
}
