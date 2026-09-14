package reviewsnapshots

import "time"

const (
	// ScopePullRequest compares the pull request base and head revisions.
	ScopePullRequest Scope = "pull-request"
	// ScopeWorkingTree compares the snapshot HEAD and captured working tree.
	ScopeWorkingTree Scope = "working-tree"
	// ScopeLastCommit compares the snapshot HEAD with its parent.
	ScopeLastCommit Scope = "last-commit"
	// ScopeCurrent returns the file's current working tree contents.
	ScopeCurrent Scope = "current"
)

const (
	// AnnotationSideOriginal targets the captured original comparison content.
	AnnotationSideOriginal AnnotationSide = "original"
	// AnnotationSideModified targets the captured modified comparison content.
	AnnotationSideModified AnnotationSide = "modified"
)

const (
	// StatusModified identifies a modified path.
	StatusModified ChangeStatus = "modified"
	// StatusAdded identifies an added path.
	StatusAdded ChangeStatus = "added"
	// StatusDeleted identifies a deleted path.
	StatusDeleted ChangeStatus = "deleted"
	// StatusRenamed identifies a renamed path.
	StatusRenamed ChangeStatus = "renamed"
)

// Scope selects the comparison represented by requested file contents.
type Scope string

// AnnotationSide identifies one side of a captured comparison.
type AnnotationSide string // @name ReviewAnnotationSide

// ChangeStatus describes how a file changed between two states.
type ChangeStatus string // @name ReviewChangeStatus

// CreateAnnotationRequest describes one immutable annotation to attach to captured content.
type CreateAnnotationRequest struct {
	FileID    string         `json:"fileId"`
	Scope     Scope          `json:"scope" enums:"working-tree,last-commit,pull-request"`
	Side      AnnotationSide `json:"side" enums:"original,modified"`
	StartLine int            `json:"startLine" minimum:"1"`
	EndLine   int            `json:"endLine" minimum:"1"`
	Summary   string         `json:"summary"`
	Rationale *string        `json:"rationale" extensions:"x-nullable" binding:"optional"`
	Author    *string        `json:"author" extensions:"x-nullable" binding:"optional"`
} // @name CreateReviewAnnotationRequest

// Annotation is an immutable agent-authored note attached to captured review content.
type Annotation struct {
	ID         string         `json:"id"`
	SnapshotID string         `json:"snapshotId"`
	FileID     string         `json:"fileId"`
	Scope      Scope          `json:"scope" enums:"working-tree,last-commit,pull-request"`
	Side       AnnotationSide `json:"side" enums:"original,modified"`
	StartLine  int            `json:"startLine"`
	EndLine    int            `json:"endLine"`
	Summary    string         `json:"summary"`
	Rationale  *string        `json:"rationale" extensions:"x-nullable"`
	Author     *string        `json:"author" extensions:"x-nullable"`
	CreatedAt  time.Time      `json:"createdAt"`
} // @name ReviewAnnotation

// AnnotationRevision identifies the current version of a snapshot's annotation collection.
type AnnotationRevision struct {
	Revision uint64 `json:"revision"`
} // @name ReviewAnnotationRevision

// FileComparison describes one file across a comparison scope.
type FileComparison struct {
	Status      ChangeStatus `json:"status"`
	OldPath     *string      `json:"oldPath" extensions:"x-nullable"`
	NewPath     *string      `json:"newPath" extensions:"x-nullable"`
	DisplayPath string       `json:"displayPath"`
	HasOriginal bool         `json:"hasOriginal"`
	HasModified bool         `json:"hasModified"`

	originalRevision        string
	modifiedRevision        string
	capturedModifiedContent *string
} // @name ReviewFileComparison

// File is a snapshot-scoped file identity and its available comparisons.
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

// FileContents contains both sides of a requested comparison.
type FileContents struct {
	OriginalContent string `json:"originalContent"`
	ModifiedContent string `json:"modifiedContent"`
} // @name ReviewFileContents

// SnapshotBranch identifies the branch captured by a snapshot.
type SnapshotBranch struct {
	Ref    string  `json:"ref"`
	Name   string  `json:"name"`
	Remote *string `json:"remote" extensions:"x-nullable"`
} // @name ReviewSnapshotBranch

// SnapshotPullRequest identifies the pull request captured by a snapshot.
type SnapshotPullRequest struct {
	Number  int    `json:"number"`
	URL     string `json:"url"`
	BaseRef string `json:"baseRef"`
	HeadRef string `json:"headRef"`
} // @name ReviewSnapshotPullRequest

// ReviewSnapshot is a detached point-in-time review resource.
type ReviewSnapshot struct {
	ID          string               `json:"id"`
	WorkspaceID string               `json:"workspaceId"`
	Branch      *SnapshotBranch      `json:"branch" extensions:"x-nullable"`
	PullRequest *SnapshotPullRequest `json:"pullRequest" extensions:"x-nullable"`
	Files       []File               `json:"files"`
	CreatedAt   time.Time            `json:"createdAt"`
} // @name ReviewSnapshot

type windowData struct {
	repoRoot    string
	branchName  string
	pullRequest *pullRequest
	files       []File
}

type pullRequest struct {
	number      int
	url         string
	baseRefName string
	headRefName string
}

type snapshotRecord struct {
	snapshot              ReviewSnapshot
	window                windowData
	annotations           []Annotation
	annotationRevision    uint64
	annotationSubscribers map[uint64]chan AnnotationRevision
	nextSubscriberID      uint64
}

type changedPath struct {
	status  ChangeStatus
	oldPath *string
	newPath *string
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
