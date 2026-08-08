package remoterepositories

// RemoteRepository is a detached provider repository snapshot.
type RemoteRepository struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	WebURL            string   `json:"webUrl"`
	CloneURL          string   `json:"cloneUrl"`
	LocalWorkspaceIDs []string `json:"localWorkspaceIds"`
} // @name RemoteRepository
