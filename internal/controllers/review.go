// NOTE: Vibecoded and not suppppppper reviewed
package controllers

// TODO: Review properly

import (
	"encoding/json"
	"net/http"

	"wade/internal/services/review"
)

type Review struct {
	review *review.Service
}

type reviewFileRequest struct {
	FileID string       `json:"fileId"`
	Scope  review.Scope `json:"scope"`
} // @name handlers.reviewFileRequest

func NewReview(reviewService *review.Service) Review {
	return Review{review: reviewService}
}

// @Summary Get review window data
// @ID getReviewWindowData
// @Tags Review
// @Produce json
// @Param project query string true "Project name"
// @Success 200 {object} review.WindowData
// @Failure 400 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Router /api/review [get]
func (h Review) GetReviewWindowData(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.URL.Query().Get("project")
	if previousSnapshotID, found := h.review.LatestSnapshotID(workspaceID); found {
		_ = h.review.DeleteSnapshot(previousSnapshotID)
	}

	snapshot, err := h.review.CreateSnapshot(r.Context(), workspaceID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	window, err := h.review.SnapshotWindowData(snapshot.ID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, window)
}

// @Summary Get review file contents
// @ID getReviewFileContents
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
	var request reviewFileRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid review file request")
		return
	}
	if request.FileID == "" || !review.IsValidScope(request.Scope) {
		writeJSONError(w, http.StatusBadRequest, "invalid review file request")
		return
	}

	workspaceID := r.URL.Query().Get("project")
	snapshotID, found := h.review.LatestSnapshotID(workspaceID)
	if !found {
		snapshot, err := h.review.CreateSnapshot(r.Context(), workspaceID)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		snapshotID = snapshot.ID
	}

	contents, err := h.review.LoadSnapshotFileContents(r.Context(), snapshotID, request.FileID, request.Scope)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, contents)
}
