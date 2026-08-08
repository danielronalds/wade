package workspaces

// Workspace is a detached local working-directory resource.
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

// WorkspaceSummary is the detached representation returned in workspace collections.
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

// WorktreeReference identifies the worktree represented by a workspace.
type WorktreeReference struct {
	ID          string `json:"id"`
	IsMain      bool   `json:"isMain"`
	IsRemovable bool   `json:"isRemovable"`
} // @name WorktreeReference

// Branch describes the branch checked out in a workspace.
type Branch struct {
	Ref        string `json:"ref"`
	Name       string `json:"name"`
	IsDetached bool   `json:"isDetached"`
	Commit     string `json:"commit"`
} // @name WorkspaceBranch

// WorkspaceLinks contains optional provider links for a workspace.
type WorkspaceLinks struct {
	Repository  *string         `json:"repository" extensions:"x-nullable"`
	PullRequest *string         `json:"pullRequest" extensions:"x-nullable"`
	Issue       *IssueReference `json:"issue" extensions:"x-nullable"`
} // @name WorkspaceLinks

// IssueReference identifies an issue associated with a workspace branch.
type IssueReference struct {
	Provider string `json:"provider"`
	Key      string `json:"key"`
	URL      string `json:"url"`
} // @name IssueReference

// WorkspaceActivity describes live terminal activity for a workspace.
type WorkspaceActivity struct {
	ActiveTerminalCount int `json:"activeTerminalCount"`
} // @name WorkspaceActivity
