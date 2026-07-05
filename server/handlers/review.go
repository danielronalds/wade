// NOTE: Vibecoded and not suppppppper reviewed
package handlers

import (
	"encoding/json"
	"net/http"

	"wade/project"
	"wade/review"
)

type Review struct {
	projects project.Store
}

type reviewFileRequest struct {
	FileID string       `json:"fileId"`
	Scope  review.Scope `json:"scope"`
}

func NewReview(projects project.Store) Review {
	return Review{projects: projects}
}

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

	writeJSON(w, http.StatusOK, data)
}

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

	data, err := review.BuildWindowData(r.Context(), projectPath)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
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
