package gitrepositories

type Repository struct {
	ID                 string   `json:"id"`
	RemoteRepositoryID *string  `json:"remoteRepositoryId"`
	MainWorkspaceID    string   `json:"mainWorkspaceId"`
	WorkspaceIDs       []string `json:"workspaceIds"`
}

type Branch struct {
	Ref        string
	Name       string
	IsDetached bool
	Commit     string
}

type Context struct {
	Repository Repository

	mainWorktreePath string
	commonDirectory  string
	remoteURL        string
	remoteIdentity   string
}

func (c Context) MainWorktreePath() string {
	return c.mainWorktreePath
}

func (c Context) CommonDirectory() string {
	return c.commonDirectory
}

func (c Context) RemoteURL() string {
	return c.remoteURL
}

func (c Context) RemoteIdentity() string {
	return c.remoteIdentity
}

type WorkspaceContext struct {
	RepositoryContext Context
	WorkspaceID       string
	WorkspacePath     string
	Branch            Branch
	IsMain            bool
	IsRemovable       bool
}
