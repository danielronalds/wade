package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"wade/internal/models/reviewsnapshots"
)

const annotationHeartbeatInterval = 15 * time.Second

// ReviewSnapshots handles review snapshot transport operations.
type ReviewSnapshots struct {
	reviewSnapshots ReviewSnapshotsModel
}

// ReviewSnapshotList is a detached workspace snapshot collection.
type ReviewSnapshotList struct {
	Items []reviewsnapshots.ReviewSnapshot `json:"items"`
} // @name ReviewSnapshotList

// AnnotationList is the authoritative annotation collection for one snapshot.
type AnnotationList struct {
	Items []reviewsnapshots.Annotation `json:"items"`
} // @name ReviewAnnotationList

type createAnnotationBody struct {
	FileID    *string                         `json:"fileId"`
	Scope     *reviewsnapshots.Scope          `json:"scope"`
	Side      *reviewsnapshots.AnnotationSide `json:"side"`
	StartLine *int                            `json:"startLine"`
	EndLine   *int                            `json:"endLine"`
	Summary   *string                         `json:"summary"`
	Rationale *string                         `json:"rationale"`
	Author    *string                         `json:"author"`
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

// List returns retained review snapshots for one configured workspace.
// @Summary List workspace review snapshots
// @Description Lists snapshots retained by the running WADE server, newest first. An empty items collection means no usable review is active; ask the reviewer to start a Review in WADE. If several snapshots exist, select an explicit snapshot ID or ask the reviewer which review is intended. Do not create another snapshot merely to obtain an ID because the browser will not be attached to it.
// @ID listWorkspaceReviewSnapshots
// @Tags Review snapshots
// @Produce json
// @Param workspaceId path string true "Workspace ID from WADE_WORKSPACE_ID"
// @Success 200 {object} ReviewSnapshotList
// @Failure 404 {object} Problem
// @Failure 500 {object} Problem
// @Router /api/v1/workspaces/{workspaceId}/review-snapshots [get]
func (controller ReviewSnapshots) List(response http.ResponseWriter, request *http.Request) {
	snapshots, err := controller.reviewSnapshots.List(request.Context(), request.PathValue("workspaceId"))
	if err != nil {
		writeModelError(response, err, "Unable to list workspace review snapshots.")
		return
	}

	writeJSON(response, http.StatusOK, ReviewSnapshotList{Items: snapshots})
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

// CreateAnnotation validates and creates an immutable agent-authored review annotation.
// @Summary Create an immutable review annotation
// @Description Create an agent note against exact captured comparison content. First run wade api list-workspace-review-snapshots --workspace-id <workspace-id> and select an explicit snapshot ID. Line numbers are one-based and startLine and endLine form an inclusive range. Revision is delete followed by create; there is no update operation.
// @ID createReviewAnnotation
// @Tags Review annotations
// @Accept json
// @Produce json
// @Param snapshotId path string true "Explicit review snapshot ID discovered with list-workspace-review-snapshots"
// @Param request body reviewsnapshots.CreateAnnotationRequest true "First discover snapshots with wade api list-workspace-review-snapshots --workspace-id <workspace-id>. JSON fields: fileId, scope, side, startLine, endLine, and summary are required; rationale and author are optional. scope is working-tree, last-commit, or pull-request. side is original or modified. Lines are one-based and inclusive. Complete invocation example: wade api create-review-annotation --snapshot-id <snapshot-id> --body @annotation.json"
// @Success 201 {object} reviewsnapshots.Annotation
// @Header 201 {string} Location "Created annotation URL used for deletion"
// @Failure 400 {object} Problem
// @Failure 404 {object} Problem
// @Failure 422 {object} Problem
// @Failure 500 {object} Problem
// @Router /api/v1/review-snapshots/{snapshotId}/annotations [post]
func (controller ReviewSnapshots) CreateAnnotation(response http.ResponseWriter, request *http.Request) {
	var body createAnnotationBody
	if err := decodeJSONBody(request, &body); err != nil {
		writeProblem(response, http.StatusBadRequest, "malformed_json", "Malformed JSON", "The request body must contain one valid review annotation and no unknown fields.")
		return
	}

	annotationRequest, complete := body.request()
	if !complete {
		writeProblem(response, http.StatusBadRequest, "malformed_json", "Malformed JSON", "fileId, scope, side, startLine, endLine, and summary are required.")
		return
	}

	annotation, err := controller.reviewSnapshots.CreateAnnotation(
		request.Context(),
		request.PathValue("snapshotId"),
		annotationRequest,
	)
	if err != nil {
		writeModelError(response, err, "Unable to create the review annotation.")
		return
	}

	location := "/api/v1/review-snapshots/" + url.PathEscape(annotation.SnapshotID) + "/annotations/" + url.PathEscape(annotation.ID)
	response.Header().Set("Location", location)
	writeJSON(response, http.StatusCreated, annotation)
}

// ListAnnotations returns the authoritative annotation collection for a snapshot.
// @Summary List review annotations
// @Description Returns annotations in creation order for the explicit review snapshot ID. Use list-workspace-review-snapshots to discover snapshots.
// @ID listReviewAnnotations
// @Tags Review annotations
// @Produce json
// @Param snapshotId path string true "Explicit review snapshot ID discovered with list-workspace-review-snapshots"
// @Success 200 {object} AnnotationList
// @Failure 404 {object} Problem
// @Failure 500 {object} Problem
// @Router /api/v1/review-snapshots/{snapshotId}/annotations [get]
func (controller ReviewSnapshots) ListAnnotations(response http.ResponseWriter, request *http.Request) {
	annotations, err := controller.reviewSnapshots.ListAnnotations(request.PathValue("snapshotId"))
	if err != nil {
		writeModelError(response, err, "Unable to list review annotations.")
		return
	}

	writeJSON(response, http.StatusOK, AnnotationList{Items: annotations})
}

// DeleteAnnotation removes one immutable annotation.
// @Summary Delete a review annotation
// @Description Deletes the annotation identified by annotationId beneath the explicit snapshotId. List annotations first if the annotation ID is not known. To revise a note, delete it and create a replacement.
// @ID deleteReviewAnnotation
// @Tags Review annotations
// @Param snapshotId path string true "Review snapshot ID"
// @Param annotationId path string true "Opaque annotation ID returned by create-review-annotation or list-review-annotations"
// @Success 204 "No Content"
// @Failure 404 {object} Problem
// @Failure 500 {object} Problem
// @Router /api/v1/review-snapshots/{snapshotId}/annotations/{annotationId} [delete]
func (controller ReviewSnapshots) DeleteAnnotation(response http.ResponseWriter, request *http.Request) {
	if err := controller.reviewSnapshots.DeleteAnnotation(
		request.PathValue("snapshotId"),
		request.PathValue("annotationId"),
	); err != nil {
		writeModelError(response, err, "Unable to delete the review annotation.")
		return
	}

	response.WriteHeader(http.StatusNoContent)
}

// AnnotationEvents streams snapshot annotation collection revisions.
// @Summary Subscribe to review annotation changes
// @Description Browser synchronisation transport. Each annotations-changed event contains the current collection revision; reload the annotation collection after every new revision.
// @ID subscribeReviewAnnotationEvents
// @Tags Review annotations
// @Produce text/event-stream
// @Param snapshotId path string true "Review snapshot ID"
// @Success 200 {string} string "Server-sent annotations-changed events"
// @Failure 404 {object} Problem
// @Failure 500 {object} Problem
// @x-wade-cli-ignore true
// @Router /api/v1/review-snapshots/{snapshotId}/annotations/events [get]
func (controller ReviewSnapshots) AnnotationEvents(response http.ResponseWriter, request *http.Request) {
	subscription, err := controller.reviewSnapshots.SubscribeAnnotations(request.PathValue("snapshotId"))
	if err != nil {
		writeModelError(response, err, "Unable to subscribe to review annotations.")
		return
	}
	defer subscription.Close()

	flusher, canFlush := response.(http.Flusher)
	if !canFlush {
		writeProblem(response, http.StatusInternalServerError, "streaming_unavailable", "Streaming unavailable", "The server cannot stream annotation changes.")
		return
	}

	response.Header().Set("Content-Type", "text/event-stream")
	response.Header().Set("Cache-Control", "no-cache")
	response.Header().Set("X-Accel-Buffering", "no")
	response.WriteHeader(http.StatusOK)
	flusher.Flush()

	heartbeat := time.NewTicker(annotationHeartbeatInterval)
	defer heartbeat.Stop()

	for {
		select {
		case revision, open := <-subscription.Updates():
			if !open {
				return
			}
			payload, marshalError := json.Marshal(revision)
			if marshalError != nil {
				return
			}
			if _, writeError := fmt.Fprintf(response, "event: annotations-changed\ndata: %s\n\n", payload); writeError != nil {
				return
			}
			flusher.Flush()
		case <-heartbeat.C:
			if _, writeError := fmt.Fprint(response, ": heartbeat\n\n"); writeError != nil {
				return
			}
			flusher.Flush()
		case <-request.Context().Done():
			return
		}
	}
}

func (body createAnnotationBody) request() (reviewsnapshots.CreateAnnotationRequest, bool) {
	if body.FileID == nil || body.Scope == nil || body.Side == nil || body.StartLine == nil || body.EndLine == nil || body.Summary == nil {
		return reviewsnapshots.CreateAnnotationRequest{}, false
	}

	return reviewsnapshots.CreateAnnotationRequest{
		FileID:    *body.FileID,
		Scope:     *body.Scope,
		Side:      *body.Side,
		StartLine: *body.StartLine,
		EndLine:   *body.EndLine,
		Summary:   *body.Summary,
		Rationale: body.Rationale,
		Author:    body.Author,
	}, true
}
