package controllers

import (
	"context"
	"net/http"

	"wade/internal/services/remoterepositories"
)

type remoteRepositoryService interface {
	List(ctx context.Context) ([]remoterepositories.RemoteRepository, error)
}

type RemoteRepositories struct {
	remoteRepositories remoteRepositoryService
}

type RemoteRepositoryList struct {
	Items []remoterepositories.RemoteRepository `json:"items"`
} // @name RemoteRepositoryList

func NewRemoteRepositories(remoteRepositories remoteRepositoryService) RemoteRepositories {
	return RemoteRepositories{remoteRepositories: remoteRepositories}
}

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
		writeServiceError(w, err, "Unable to list remote repositories.")
		return
	}

	writeJSON(w, http.StatusOK, RemoteRepositoryList{Items: remoteRepositories})
}
