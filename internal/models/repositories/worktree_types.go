package repositories

// Worktree is a detached local Git worktree resource.
type Worktree struct {
	ID                      string   `json:"id"`
	RepositoryID            string   `json:"repositoryId"`
	WorkspaceID             string   `json:"workspaceId"`
	Name                    string   `json:"name"`
	Branch                  *Branch  `json:"branch" extensions:"x-nullable"`
	IsMain                  bool     `json:"isMain"`
	IsRemovable             bool     `json:"isRemovable"`
	IgnoredFileCopyWarnings []string `json:"ignoredFileCopyWarnings,omitempty"`

	path string
} // @name Worktree

// Branch is a detached local or remote Git branch resource.
type Branch struct {
	Ref                   string  `json:"ref"`
	Name                  string  `json:"name"`
	Remote                *string `json:"remote" extensions:"x-nullable"`
	HasLocalBranch        bool    `json:"hasLocalBranch"`
	CheckedOutWorkspaceID *string `json:"checkedOutWorkspaceId" extensions:"x-nullable"`
} // @name Branch

// BranchKind selects local or remote branch discovery.
type BranchKind string

// Supported branch discovery kinds.
const (
	BranchKindLocal  BranchKind = "local"
	BranchKindRemote BranchKind = "remote"
)
