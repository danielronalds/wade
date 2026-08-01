package workspaces

type Workspace struct {
	ID                 string             `json:"id"`
	Name               string             `json:"name"`
	RepositoryID       *string            `json:"repositoryId"`
	RemoteRepositoryID *string            `json:"remoteRepositoryId"`
	Worktree           *WorktreeReference `json:"worktree"`
	Branch             *Branch            `json:"branch"`
	Links              WorkspaceLinks     `json:"links"`
	Activity           WorkspaceActivity  `json:"activity"`
}

type WorkspaceSummary struct {
	ID                 string             `json:"id"`
	Name               string             `json:"name"`
	RepositoryID       *string            `json:"repositoryId"`
	RemoteRepositoryID *string            `json:"remoteRepositoryId"`
	Worktree           *WorktreeReference `json:"worktree"`
	Branch             *Branch            `json:"branch"`
	Links              WorkspaceLinks     `json:"links"`
	Activity           WorkspaceActivity  `json:"activity"`
}

type WorktreeReference struct {
	ID          string `json:"id"`
	IsMain      bool   `json:"isMain"`
	IsRemovable bool   `json:"isRemovable"`
}

type Branch struct {
	Ref        string `json:"ref"`
	Name       string `json:"name"`
	IsDetached bool   `json:"isDetached"`
	Commit     string `json:"commit"`
}

type WorkspaceLinks struct {
	Repository  *string         `json:"repository"`
	PullRequest *string         `json:"pullRequest"`
	Issue       *IssueReference `json:"issue"`
}

type IssueReference struct {
	Provider string `json:"provider"`
	Key      string `json:"key"`
	URL      string `json:"url"`
}

type WorkspaceActivity struct {
	ActiveTerminalCount int `json:"activeTerminalCount"`
}
