package review

// TODO: Review properly

import "context"

const (
	ScopePullRequest Scope = "pull-request"
	ScopeGitDiff     Scope = "git-diff"
	ScopeLastCommit  Scope = "last-commit"
	ScopeAllFiles    Scope = "all-files"
)

const (
	StatusModified ChangeStatus = "modified"
	StatusAdded    ChangeStatus = "added"
	StatusDeleted  ChangeStatus = "deleted"
	StatusRenamed  ChangeStatus = "renamed"
)

type Scope string

type ChangeStatus string

type PullRequest struct {
	Number      int    `json:"number"`
	URL         string `json:"url"`
	BaseRefName string `json:"baseRefName"`
	HeadRefName string `json:"headRefName"`
}

type FileComparison struct {
	Status      ChangeStatus `json:"status"`
	OldPath     *string      `json:"oldPath"`
	NewPath     *string      `json:"newPath"`
	DisplayPath string       `json:"displayPath"`
	HasOriginal bool         `json:"hasOriginal"`
	HasModified bool         `json:"hasModified"`

	originalRevision string
	modifiedRevision string
}

type File struct {
	ID                 string          `json:"id"`
	Path               string          `json:"path"`
	WorktreeStatus     *ChangeStatus   `json:"worktreeStatus"`
	HasWorkingTreeFile bool            `json:"hasWorkingTreeFile"`
	InGitDiff          bool            `json:"inGitDiff"`
	InLastCommit       bool            `json:"inLastCommit"`
	InPullRequest      bool            `json:"inPullRequest"`
	GitDiff            *FileComparison `json:"gitDiff"`
	LastCommit         *FileComparison `json:"lastCommit"`
	PullRequest        *FileComparison `json:"pullRequest"`
}

type WindowData struct {
	RepoRoot    string       `json:"repoRoot"`
	BranchName  string       `json:"branchName"`
	PullRequest *PullRequest `json:"pullRequest"`
	Files       []File       `json:"files"`
}

type FileContents struct {
	OriginalContent string `json:"originalContent"`
	ModifiedContent string `json:"modifiedContent"`
}

type Service struct {
	git    gitRepository
	github gitHubRepository
	files  fileRepository
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
