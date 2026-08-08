package controllers

import "net/http"

// Repositories serves local repository resources.
type Repositories struct {
	repositories RepositoriesModel
}

// NewRepositories constructs the Repositories controller.
func NewRepositories(repositories RepositoriesModel) Repositories {
	return Repositories{repositories: repositories}
}

// @Summary Get a local repository
// @ID getRepository
// @Tags Repositories
// @Produce json
// @Param repositoryId path string true "Repository ID"
// @Success 200 {object} repositories.Repository
// @Failure 404 {object} Problem
// @Failure 409 {object} Problem
// @Failure 422 {object} Problem
// @Failure 500 {object} Problem
// @Router /api/v1/repositories/{repositoryId} [get]
func (h Repositories) Get(w http.ResponseWriter, r *http.Request) {
	repository, err := h.repositories.Get(r.Context(), r.PathValue("repositoryId"))
	if err != nil {
		writeModelError(w, err, "Unable to load the repository.")
		return
	}
	writeJSON(w, http.StatusOK, repository)
}
