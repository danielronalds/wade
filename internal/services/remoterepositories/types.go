package remoterepositories

// TODO: Review properly

type RemoteRepository struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	WebURL            string   `json:"webUrl"`
	CloneURL          string   `json:"cloneUrl"`
	LocalWorkspaceIDs []string `json:"localWorkspaceIds"`
} // @name RemoteRepository

type WorkspaceDirectory struct {
	Setting string
	Path    string
}

type CloneRequest struct {
	RemoteRepositoryID string
	WorkspaceDirectory string
}
