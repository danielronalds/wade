// NOTE: Vibecoded and not suppppppper reviewed
package worktree

type Worktree struct {
	Name        string `json:"name"`
	ProjectName string `json:"projectName"`
	Path        string `json:"path"`
	Branch      string `json:"branch"`
	IsBase      bool   `json:"isBase"`
	IsCurrent   bool   `json:"isCurrent"`
	IsRemovable bool   `json:"isRemovable"`
}

type RemoteBranchList struct {
	Remote   string         `json:"remote"`
	Branches []RemoteBranch `json:"branches"`
}

type RemoteBranch struct {
	Name                string `json:"name"`
	Branch              string `json:"branch"`
	HasLocalBranch      bool   `json:"hasLocalBranch"`
	IsCheckedOut        bool   `json:"isCheckedOut"`
	WorktreeName        string `json:"worktreeName"`
	WorktreeProjectName string `json:"worktreeProjectName"`
}
