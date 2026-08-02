package controllers

import (
	"context"
	"net/http"

	"wade/internal/services/gitrepositories"
)

type localRepositoryService interface {
	Resolve(ctx context.Context, repositoryID string) (gitrepositories.Context, error)
}

type Repositories struct {
	repositories localRepositoryService
}

func NewRepositories(repositories localRepositoryService) Repositories {
	return Repositories{repositories: repositories}
}

// @Summary Get a local repository
// @ID getRepository
// @Tags Repositories
// @Produce json
// @Param repositoryId path string true "Repository ID"
// @Success 200 {object} gitrepositories.Repository
// @Failure 404 {object} Problem
// @Failure 409 {object} Problem
// @Failure 422 {object} Problem
// @Failure 500 {object} Problem
// @Router /api/v1/repositories/{repositoryId} [get]
func (h Repositories) Get(w http.ResponseWriter, r *http.Request) {
	repository, err := h.repositories.Resolve(r.Context(), r.PathValue("repositoryId"))
	if err != nil {
		writeServiceError(w, err, "Unable to load the repository.")
		return
	}

	writeJSON(w, http.StatusOK, repository.Repository)
}
