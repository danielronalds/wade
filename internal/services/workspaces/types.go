package workspaces

type Workspace struct {
	ID                 string             `json:"id"`
	Name               string             `json:"name"`
	RepositoryID       *string            `json:"repositoryId" extensions:"x-nullable"`
	RemoteRepositoryID *string            `json:"remoteRepositoryId" extensions:"x-nullable"`
	Worktree           *WorktreeReference `json:"worktree" extensions:"x-nullable"`
	Branch             *Branch            `json:"branch" extensions:"x-nullable"`
	Links              WorkspaceLinks     `json:"links"`
	Activity           WorkspaceActivity  `json:"activity"`
} // @name Workspace

type WorkspaceSummary struct {
	ID                 string             `json:"id"`
	Name               string             `json:"name"`
	RepositoryID       *string            `json:"repositoryId" extensions:"x-nullable"`
	RemoteRepositoryID *string            `json:"remoteRepositoryId" extensions:"x-nullable"`
	Worktree           *WorktreeReference `json:"worktree" extensions:"x-nullable"`
	Branch             *Branch            `json:"branch" extensions:"x-nullable"`
	Links              WorkspaceLinks     `json:"links"`
	Activity           WorkspaceActivity  `json:"activity"`
} // @name WorkspaceSummary

type WorktreeReference struct {
	ID          string `json:"id"`
	IsMain      bool   `json:"isMain"`
	IsRemovable bool   `json:"isRemovable"`
} // @name WorktreeReference

type Branch struct {
	Ref        string `json:"ref"`
	Name       string `json:"name"`
	IsDetached bool   `json:"isDetached"`
	Commit     string `json:"commit"`
} // @name WorkspaceBranch

type WorkspaceLinks struct {
	Repository  *string         `json:"repository" extensions:"x-nullable"`
	PullRequest *string         `json:"pullRequest" extensions:"x-nullable"`
	Issue       *IssueReference `json:"issue" extensions:"x-nullable"`
} // @name WorkspaceLinks

type IssueReference struct {
	Provider string `json:"provider"`
	Key      string `json:"key"`
	URL      string `json:"url"`
} // @name IssueReference

type WorkspaceActivity struct {
	ActiveTerminalCount int `json:"activeTerminalCount"`
} // @name WorkspaceActivity
