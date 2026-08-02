// NOTE: Vibecoded and not suppppppper reviewed
package worktrees

// TODO: Review properly

type Worktree struct {
	ID                      string   `json:"id"`
	RepositoryID            string   `json:"repositoryId"`
	WorkspaceID             string   `json:"workspaceId"`
	Name                    string   `json:"name"`
	Branch                  *Branch  `json:"branch"`
	IsMain                  bool     `json:"isMain"`
	IsRemovable             bool     `json:"isRemovable"`
	IgnoredFileCopyWarnings []string `json:"ignoredFileCopyWarnings,omitempty"`

	path               string
	workspaceDirectory string
}

func (w Worktree) Path() string {
	return w.path
}

type Branch struct {
	Ref                   string  `json:"ref"`
	Name                  string  `json:"name"`
	Remote                *string `json:"remote"`
	HasLocalBranch        bool    `json:"hasLocalBranch"`
	CheckedOutWorkspaceID *string `json:"checkedOutWorkspaceId"`
}

type BranchKind string

const (
	BranchKindLocal  BranchKind = "local"
	BranchKindRemote BranchKind = "remote"
)
