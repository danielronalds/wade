// NOTE: Vibecoded and not suppppppper reviewed
package worktrees

// TODO: Review properly

type Worktree struct {
	Name                    string   `json:"name"`
	ProjectName             string   `json:"projectName"`
	Path                    string   `json:"path"`
	Branch                  string   `json:"branch"`
	IsBase                  bool     `json:"isBase"`
	IsCurrent               bool     `json:"isCurrent"`
	IsRemovable             bool     `json:"isRemovable"`
	IgnoredFileCopyWarnings []string `json:"ignoredFileCopyWarnings,omitempty"`
} // @name worktree.Worktree

type RemoteBranchList struct {
	Remote   string         `json:"remote"`
	Branches []RemoteBranch `json:"branches"`
} // @name worktree.RemoteBranchList

type RemoteBranch struct {
	Name                string `json:"name"`
	Branch              string `json:"branch"`
	HasLocalBranch      bool   `json:"hasLocalBranch"`
	IsCheckedOut        bool   `json:"isCheckedOut"`
	WorktreeName        string `json:"worktreeName"`
	WorktreeProjectName string `json:"worktreeProjectName"`
} // @name worktree.RemoteBranch
