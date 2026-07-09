// NOTE: Vibecoded and not suppppppper reviewed
package handlers

import (
	"encoding/json"
	"net/http"
	"sync"

	"wade/internal/project"
	"wade/internal/review"
)

type Review struct {
	projects project.Store
	cache    *reviewWindowCache
}

type reviewWindowCache struct {
	mu    sync.RWMutex
	items map[string]review.WindowData
}

type reviewFileRequest struct {
	FileID string       `json:"fileId"`
	Scope  review.Scope `json:"scope"`
}

func NewReview(projects project.Store) Review {
	return Review{
		projects: projects,
		cache: &reviewWindowCache{
			items: make(map[string]review.WindowData),
		},
	}
}

// @Summary Get review window data
// @Tags Review
// @Produce json
// @Param project query string true "Project name"
// @Success 200 {object} review.WindowData
// @Failure 400 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Router /api/review [get]
func (h Review) GetReviewWindowData(w http.ResponseWriter, r *http.Request) {
	projectPath, ok := resolveProjectPath(w, r, h.projects)
	if !ok {
		return
	}

	data, err := review.BuildWindowData(r.Context(), projectPath)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.cache.set(projectPath, data)
	writeJSON(w, http.StatusOK, data)
}

// @Summary Get review file contents
// @Tags Review
// @Accept json
// @Produce json
// @Param project query string true "Project name"
// @Param request body reviewFileRequest true "Review file request"
// @Success 200 {object} review.FileContents
// @Failure 400 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Router /api/review/file [post]
func (h Review) GetReviewFileContents(w http.ResponseWriter, r *http.Request) {
	projectPath, ok := resolveProjectPath(w, r, h.projects)
	if !ok {
		return
	}

	var request reviewFileRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid review file request")
		return
	}

	if request.FileID == "" || !review.IsValidScope(request.Scope) {
		writeJSONError(w, http.StatusBadRequest, "invalid review file request")
		return
	}

	data, found := h.cache.get(projectPath)
	if !found {
		freshData, err := review.BuildWindowData(r.Context(), projectPath)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}

		data = freshData
		h.cache.set(projectPath, data)
	}

	file, found := findReviewFile(data.Files, request.FileID)
	if !found {
		writeJSONError(w, http.StatusNotFound, "review file not found")
		return
	}

	contents, err := review.LoadFileContents(r.Context(), data.RepoRoot, file, request.Scope)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, contents)
}

func (c *reviewWindowCache) get(projectPath string) (review.WindowData, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	data, found := c.items[projectPath]
	return data, found
}

func (c *reviewWindowCache) set(projectPath string, data review.WindowData) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[projectPath] = data
}

func resolveProjectPath(w http.ResponseWriter, r *http.Request, projects project.Store) (string, bool) {
	projectPath, err := projects.Path(r.URL.Query().Get("project"))
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "project not found")
		return "", false
	}

	return projectPath, true
}

func findReviewFile(files []review.File, fileID string) (review.File, bool) {
	for _, file := range files {
		if file.ID == fileID {
			return file, true
		}
	}

	return review.File{}, false
}
