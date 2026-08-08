package repositories

// Repository is a detached local Git repository snapshot.
type Repository struct {
	ID                 string   `json:"id"`
	RemoteRepositoryID *string  `json:"remoteRepositoryId" extensions:"x-nullable"`
	MainWorkspaceID    string   `json:"mainWorkspaceId"`
	WorkspaceIDs       []string `json:"workspaceIds"`
} // @name Repository

// WorkspaceBranch describes the branch checked out by one workspace.
type WorkspaceBranch struct {
	Ref        string
	Name       string
	IsDetached bool
	Commit     string
}

// WorkspaceContext is the detached Git context used to enrich a workspace.
type WorkspaceContext struct {
	WorkspaceID string
	Repository  Repository
	Branch      WorkspaceBranch
	IsMain      bool
	IsRemovable bool
}

type repositoryContext struct {
	repository       Repository
	mainWorktreePath string
	commonDirectory  string
	remoteURL        string
	remoteIdentity   string
}

func cloneRepository(repository Repository) Repository {
	cloned := repository
	cloned.WorkspaceIDs = append([]string(nil), repository.WorkspaceIDs...)
	if repository.RemoteRepositoryID != nil {
		remoteRepositoryID := *repository.RemoteRepositoryID
		cloned.RemoteRepositoryID = &remoteRepositoryID
	}
	return cloned
}
