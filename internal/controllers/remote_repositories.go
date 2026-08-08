package controllers

import (
	"net/http"

	"wade/internal/models/remoterepositories"
)

// RemoteRepositories composes provider repositories with local workspace IDs.
type RemoteRepositories struct {
	remoteRepositories RemoteRepositoriesModel
	repositories       RepositoriesModel
}

// RemoteRepositoryList is the collection response for provider repositories.
type RemoteRepositoryList struct {
	Items []remoterepositories.RemoteRepository `json:"items"`
} // @name RemoteRepositoryList

// NewRemoteRepositories constructs the RemoteRepositories controller.
func NewRemoteRepositories(remoteRepositories RemoteRepositoriesModel, repositories RepositoriesModel) RemoteRepositories {
	return RemoteRepositories{remoteRepositories: remoteRepositories, repositories: repositories}
}

// List returns provider repositories enriched with local workspace identities.
// @Summary List remote repositories
// @ID listRemoteRepositories
// @Tags Remote repositories
// @Produce json
// @Success 200 {object} RemoteRepositoryList
// @Failure 500 {object} Problem
// @Router /api/v1/remote-repositories [get]
func (h RemoteRepositories) List(w http.ResponseWriter, r *http.Request) {
	remoteRepositories, err := h.remoteRepositories.List(r.Context())
	if err != nil {
		writeModelError(w, err, "Unable to list remote repositories.")
		return
	}

	remoteRepositoryIDs := make([]string, 0, len(remoteRepositories))
	for _, repository := range remoteRepositories {
		remoteRepositoryIDs = append(remoteRepositoryIDs, repository.ID)
	}
	workspaceIDs, err := h.repositories.WorkspaceIDsByRemoteRepository(r.Context(), remoteRepositoryIDs)
	if err != nil {
		writeModelError(w, err, "Unable to map local workspaces to remote repositories.")
		return
	}
	for index := range remoteRepositories {
		ids := workspaceIDs[remoteRepositories[index].ID]
		if ids == nil {
			ids = []string{}
		}
		remoteRepositories[index].LocalWorkspaceIDs = append([]string(nil), ids...)
	}

	writeJSON(w, http.StatusOK, RemoteRepositoryList{Items: remoteRepositories})
}
