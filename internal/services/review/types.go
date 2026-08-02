package review

// TODO: Review properly

import (
	"context"
	"sync"
	"time"
)

const (
	ScopePullRequest Scope = "pull-request"
	ScopeWorkingTree Scope = "working-tree"
	ScopeLastCommit  Scope = "last-commit"
	ScopeCurrent     Scope = "current"
)

const (
	StatusModified ChangeStatus = "modified"
	StatusAdded    ChangeStatus = "added"
	StatusDeleted  ChangeStatus = "deleted"
	StatusRenamed  ChangeStatus = "renamed"
)

type Scope string

type ChangeStatus string // @name ReviewChangeStatus

type PullRequest struct {
	Number      int    `json:"number"`
	URL         string `json:"url"`
	BaseRefName string `json:"baseRefName"`
	HeadRefName string `json:"headRefName"`
} // @name PullRequest

type FileComparison struct {
	Status      ChangeStatus `json:"status"`
	OldPath     *string      `json:"oldPath" extensions:"x-nullable"`
	NewPath     *string      `json:"newPath" extensions:"x-nullable"`
	DisplayPath string       `json:"displayPath"`
	HasOriginal bool         `json:"hasOriginal"`
	HasModified bool         `json:"hasModified"`

	originalRevision string
	modifiedRevision string
} // @name ReviewFileComparison

type File struct {
	ID                 string          `json:"id"`
	Path               string          `json:"path"`
	WorktreeStatus     *ChangeStatus   `json:"worktreeStatus" extensions:"x-nullable"`
	HasWorkingTreeFile bool            `json:"hasWorkingTreeFile"`
	InGitDiff          bool            `json:"inGitDiff"`
	InLastCommit       bool            `json:"inLastCommit"`
	InPullRequest      bool            `json:"inPullRequest"`
	GitDiff            *FileComparison `json:"gitDiff" extensions:"x-nullable"`
	LastCommit         *FileComparison `json:"lastCommit" extensions:"x-nullable"`
	PullRequest        *FileComparison `json:"pullRequest" extensions:"x-nullable"`
} // @name ReviewFile

type WindowData struct {
	RepoRoot    string       `json:"repoRoot"`
	BranchName  string       `json:"branchName"`
	PullRequest *PullRequest `json:"pullRequest"`
	Files       []File       `json:"files"`
}

type FileContents struct {
	OriginalContent string `json:"originalContent"`
	ModifiedContent string `json:"modifiedContent"`
} // @name ReviewFileContents

type SnapshotBranch struct {
	Ref    string  `json:"ref"`
	Name   string  `json:"name"`
	Remote *string `json:"remote" extensions:"x-nullable"`
} // @name ReviewSnapshotBranch

type SnapshotPullRequest struct {
	Number  int    `json:"number"`
	URL     string `json:"url"`
	BaseRef string `json:"baseRef"`
	HeadRef string `json:"headRef"`
} // @name ReviewSnapshotPullRequest

type ReviewSnapshot struct {
	ID          string               `json:"id"`
	WorkspaceID string               `json:"workspaceId"`
	Branch      *SnapshotBranch      `json:"branch" extensions:"x-nullable"`
	PullRequest *SnapshotPullRequest `json:"pullRequest" extensions:"x-nullable"`
	Files       []File               `json:"files"`
	CreatedAt   time.Time            `json:"createdAt"`
} // @name ReviewSnapshot

type Service struct {
	workspaces WorkspaceRepository
	git        gitRepository
	github     gitHubRepository
	files      fileRepository
	state      *snapshotState
}

type WorkspaceRepository interface {
	Path(workspaceID string) (string, error)
}

type snapshotState struct {
	mu    sync.RWMutex
	items map[string]snapshotRecord
}

type snapshotRecord struct {
	snapshot      ReviewSnapshot
	workspacePath string
	window        WindowData
}

type changedPath struct {
	status  ChangeStatus
	oldPath *string
	newPath *string
}

type pullRequestResponse struct {
	Number      int    `json:"number"`
	URL         string `json:"url"`
	State       string `json:"state"`
	BaseRefName string `json:"baseRefName"`
	HeadRefName string `json:"headRefName"`
}

type fileSeed struct {
	path               string
	worktreeStatus     *ChangeStatus
	hasWorkingTreeFile bool
	inGitDiff          bool
	inLastCommit       bool
	inPullRequest      bool
	gitDiff            *FileComparison
	lastCommit         *FileComparison
	pullRequest        *FileComparison
}

type gitRepository interface {
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

type gitHubRepository interface {
	PullRequest(ctx context.Context, repoRoot string, branch string) ([]byte, error)
}

type fileRepository interface {
	ReadFile(path string) ([]byte, error)
}
