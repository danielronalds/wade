// NOTE: Vibecoded and not suppppppper reviewed
package repositories

// TODO: Review properly

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

type Branch struct {
	Ref                   string  `json:"ref"`
	Name                  string  `json:"name"`
	Remote                *string `json:"remote" extensions:"x-nullable"`
	HasLocalBranch        bool    `json:"hasLocalBranch"`
	CheckedOutWorkspaceID *string `json:"checkedOutWorkspaceId" extensions:"x-nullable"`
} // @name Branch

type BranchKind string

const (
	BranchKindLocal  BranchKind = "local"
	BranchKindRemote BranchKind = "remote"
)
