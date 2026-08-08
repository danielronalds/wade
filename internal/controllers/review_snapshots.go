package controllers

import (
	"net/http"
	"net/url"

	"wade/internal/models/reviewsnapshots"
)

// ReviewSnapshots handles review snapshot transport operations.
type ReviewSnapshots struct {
	reviewSnapshots ReviewSnapshotsModel
}

// NewReviewSnapshots constructs the ReviewSnapshots controller.
func NewReviewSnapshots(reviewSnapshots ReviewSnapshotsModel) ReviewSnapshots {
	return ReviewSnapshots{reviewSnapshots: reviewSnapshots}
}

// Create captures and returns a review snapshot for a workspace.
// @Summary Create a review snapshot
// @ID createReviewSnapshot
// @Tags Review snapshots
// @Produce json
// @Param workspaceId path string true "Workspace ID"
// @Success 201 {object} reviewsnapshots.ReviewSnapshot
// @Header 201 {string} Location "Created review snapshot URL"
// @Failure 404 {object} Problem
// @Failure 422 {object} Problem
// @Failure 500 {object} Problem
// @Router /api/v1/workspaces/{workspaceId}/review-snapshots [post]
func (controller ReviewSnapshots) Create(response http.ResponseWriter, request *http.Request) {
	snapshot, err := controller.reviewSnapshots.Create(request.Context(), request.PathValue("workspaceId"))
	if err != nil {
		writeModelError(response, err, "Unable to create the review snapshot.")
		return
	}

	response.Header().Set("Location", "/api/v1/review-snapshots/"+url.PathEscape(snapshot.ID))
	writeJSON(response, http.StatusCreated, snapshot)
}

// Get returns a previously captured review snapshot.
// @Summary Get a review snapshot
// @ID getReviewSnapshot
// @Tags Review snapshots
// @Produce json
// @Param snapshotId path string true "Review snapshot ID"
// @Success 200 {object} reviewsnapshots.ReviewSnapshot
// @Failure 404 {object} Problem
// @Failure 500 {object} Problem
// @Router /api/v1/review-snapshots/{snapshotId} [get]
func (controller ReviewSnapshots) Get(response http.ResponseWriter, request *http.Request) {
	snapshot, err := controller.reviewSnapshots.Get(request.PathValue("snapshotId"))
	if err != nil {
		writeModelError(response, err, "Unable to load the review snapshot.")
		return
	}

	writeJSON(response, http.StatusOK, snapshot)
}

// GetFileContents returns one snapshot file comparison.
// @Summary Get review snapshot file contents
// @ID getReviewSnapshotFileContents
// @Tags Review snapshots
// @Produce json
// @Param snapshotId path string true "Review snapshot ID"
// @Param fileId path string true "Snapshot file ID"
// @Param scope query string true "Comparison scope" Enums(pull-request,working-tree,last-commit,current)
// @Success 200 {object} reviewsnapshots.FileContents
// @Failure 404 {object} Problem
// @Failure 422 {object} Problem
// @Failure 500 {object} Problem
// @Router /api/v1/review-snapshots/{snapshotId}/files/{fileId}/contents [get]
func (controller ReviewSnapshots) GetFileContents(response http.ResponseWriter, request *http.Request) {
	contents, err := controller.reviewSnapshots.FileContents(
		request.Context(),
		request.PathValue("snapshotId"),
		request.PathValue("fileId"),
		reviewsnapshots.Scope(request.URL.Query().Get("scope")),
	)
	if err != nil {
		writeModelError(response, err, "Unable to load review file contents.")
		return
	}

	writeJSON(response, http.StatusOK, contents)
}

// Delete removes an in-memory review snapshot.
// @Summary Delete a review snapshot
// @ID deleteReviewSnapshot
// @Tags Review snapshots
// @Param snapshotId path string true "Review snapshot ID"
// @Success 204 "No Content"
// @Failure 404 {object} Problem
// @Failure 500 {object} Problem
// @Router /api/v1/review-snapshots/{snapshotId} [delete]
func (controller ReviewSnapshots) Delete(response http.ResponseWriter, request *http.Request) {
	if err := controller.reviewSnapshots.Delete(request.PathValue("snapshotId")); err != nil {
		writeModelError(response, err, "Unable to delete the review snapshot.")
		return
	}

	response.WriteHeader(http.StatusNoContent)
}
