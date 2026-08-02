package controllers

import (
	"net/http"
	"net/url"

	"wade/internal/services/review"
)

type ReviewSnapshots struct {
	review *review.Service
}

func NewReviewSnapshots(reviewService *review.Service) ReviewSnapshots {
	return ReviewSnapshots{review: reviewService}
}

// @Summary Create a review snapshot
// @ID createReviewSnapshot
// @Tags Review snapshots
// @Produce json
// @Param workspaceId path string true "Workspace ID"
// @Success 201 {object} review.ReviewSnapshot
// @Header 201 {string} Location "Created review snapshot URL"
// @Failure 404 {object} Problem
// @Failure 422 {object} Problem
// @Failure 500 {object} Problem
// @Router /api/v1/workspaces/{workspaceId}/review-snapshots [post]
func (h ReviewSnapshots) Create(w http.ResponseWriter, r *http.Request) {
	snapshot, err := h.review.CreateSnapshot(r.Context(), r.PathValue("workspaceId"))
	if err != nil {
		writeServiceError(w, err, "Unable to create the review snapshot.")
		return
	}

	w.Header().Set("Location", "/api/v1/review-snapshots/"+url.PathEscape(snapshot.ID))
	writeJSON(w, http.StatusCreated, snapshot)
}

// @Summary Get a review snapshot
// @ID getReviewSnapshot
// @Tags Review snapshots
// @Produce json
// @Param snapshotId path string true "Review snapshot ID"
// @Success 200 {object} review.ReviewSnapshot
// @Failure 404 {object} Problem
// @Failure 500 {object} Problem
// @Router /api/v1/review-snapshots/{snapshotId} [get]
func (h ReviewSnapshots) Get(w http.ResponseWriter, r *http.Request) {
	snapshot, err := h.review.GetSnapshot(r.PathValue("snapshotId"))
	if err != nil {
		writeServiceError(w, err, "Unable to load the review snapshot.")
		return
	}

	writeJSON(w, http.StatusOK, snapshot)
}

// @Summary Get review snapshot file contents
// @ID getReviewSnapshotFileContents
// @Tags Review snapshots
// @Produce json
// @Param snapshotId path string true "Review snapshot ID"
// @Param fileId path string true "Snapshot file ID"
// @Param scope query string true "Comparison scope" Enums(pull-request,working-tree,last-commit,current)
// @Success 200 {object} review.FileContents
// @Failure 404 {object} Problem
// @Failure 422 {object} Problem
// @Failure 500 {object} Problem
// @Router /api/v1/review-snapshots/{snapshotId}/files/{fileId}/contents [get]
func (h ReviewSnapshots) GetFileContents(w http.ResponseWriter, r *http.Request) {
	scope := review.Scope(r.URL.Query().Get("scope"))
	if !review.IsValidScope(scope) {
		writeServiceError(w, review.InvalidScopeError{Scope: scope}, "Unable to load review file contents.")
		return
	}

	contents, err := h.review.LoadSnapshotFileContents(
		r.Context(),
		r.PathValue("snapshotId"),
		r.PathValue("fileId"),
		scope,
	)
	if err != nil {
		writeServiceError(w, err, "Unable to load review file contents.")
		return
	}

	writeJSON(w, http.StatusOK, contents)
}

// @Summary Delete a review snapshot
// @ID deleteReviewSnapshot
// @Tags Review snapshots
// @Param snapshotId path string true "Review snapshot ID"
// @Success 204 "No Content"
// @Failure 404 {object} Problem
// @Failure 500 {object} Problem
// @Router /api/v1/review-snapshots/{snapshotId} [delete]
func (h ReviewSnapshots) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.review.DeleteSnapshot(r.PathValue("snapshotId")); err != nil {
		writeServiceError(w, err, "Unable to delete the review snapshot.")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
