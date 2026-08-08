package reviewsnapshots

import (
	"context"

	"wade/internal/infrastructure/github"
)

// WorkspaceDiscovery resolves configured workspace identities to local paths.
type WorkspaceDiscovery interface {
	Resolve(workspaceID string) (string, bool, error)
}

// Git provides cohesive repository inspection and revision-content operations.
type Git interface {
	RepoRoot(ctx context.Context, cwd string) (string, error)
	VerifyHead(ctx context.Context, repoRoot string) error
	ReviewCurrentBranch(ctx context.Context, repoRoot string) ([]byte, error)
	TrackedDiffNameStatus(ctx context.Context, repoRoot string) ([]byte, error)
	UntrackedFiles(ctx context.Context, repoRoot string) ([]byte, error)
	TrackedFiles(ctx context.Context, repoRoot string) ([]byte, error)
	DeletedFiles(ctx context.Context, repoRoot string) ([]byte, error)
	LastCommitNameStatus(ctx context.Context, repoRoot string) ([]byte, error)
	DiffNameStatusBetween(ctx context.Context, repoRoot string, originalRevision string, modifiedRevision string) ([]byte, error)
	CommitRevision(ctx context.Context, repoRoot string, revision string) ([]byte, error)
	MergeBase(ctx context.Context, repoRoot string, revision string) ([]byte, error)
	RevisionContent(ctx context.Context, repoRoot string, revision string, filePath string) ([]byte, error)
}

// GitHub resolves pull request metadata for a local repository branch.
type GitHub interface {
	PullRequest(ctx context.Context, repoRoot string, branch string) (*github.PullRequest, error)
}

// FileSystem provides working tree file reads.
type FileSystem interface {
	ReadFile(path string) ([]byte, error)
}
