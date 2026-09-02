package reviewsnapshots

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	maximumAnnotationsPerSnapshot   = 500
	maximumAnnotationSummaryBytes   = 4_096
	maximumAnnotationRationaleBytes = 16_384
	maximumAnnotationAuthorBytes    = 128
)

// AnnotationSubscription is a live, snapshot-scoped annotation revision stream.
type AnnotationSubscription interface {
	Updates() <-chan AnnotationRevision
	Close()
}

type annotationSubscription struct {
	updates <-chan AnnotationRevision
	close   func()
	once    sync.Once
}

// CreateAnnotation validates and appends an immutable annotation to captured snapshot content.
func (model *Model) CreateAnnotation(
	ctx context.Context,
	snapshotID string,
	request CreateAnnotationRequest,
) (Annotation, error) {
	normalisedRequest, err := normaliseAnnotationRequest(request)
	if err != nil {
		return Annotation{}, err
	}
	if err := ctx.Err(); err != nil {
		return Annotation{}, err
	}

	model.mu.RLock()
	record, found := model.items[snapshotID]
	if found && len(record.annotations) >= maximumAnnotationsPerSnapshot {
		model.mu.RUnlock()
		return Annotation{}, InvalidAnnotationError{Detail: "a review snapshot may contain at most 500 annotations"}
	}
	model.mu.RUnlock()
	if !found {
		return Annotation{}, SnapshotNotFoundError{SnapshotID: snapshotID}
	}

	var file *File
	for index := range record.window.files {
		if record.window.files[index].ID == normalisedRequest.FileID {
			file = &record.window.files[index]
			break
		}
	}
	if file == nil {
		return Annotation{}, SnapshotFileNotFoundError{SnapshotID: snapshotID, FileID: normalisedRequest.FileID}
	}

	var comparison *FileComparison
	switch normalisedRequest.Scope {
	case ScopeWorkingTree:
		comparison = file.GitDiff
	case ScopeLastCommit:
		comparison = file.LastCommit
	case ScopePullRequest:
		comparison = file.PullRequest
	}
	if comparison == nil {
		return Annotation{}, InvalidAnnotationError{Detail: "fileId does not belong to the selected scope"}
	}
	if normalisedRequest.Side == AnnotationSideOriginal && !comparison.HasOriginal {
		return Annotation{}, InvalidAnnotationError{Detail: "the original side does not exist for the selected file and scope"}
	}
	if normalisedRequest.Side == AnnotationSideModified && !comparison.HasModified {
		return Annotation{}, InvalidAnnotationError{Detail: "the modified side does not exist for the selected file and scope"}
	}

	contents, err := model.loadFileContents(ctx, record.window.repoRoot, *file, normalisedRequest.Scope)
	if err != nil {
		return Annotation{}, err
	}
	selectedContent := contents.ModifiedContent
	if normalisedRequest.Side == AnnotationSideOriginal {
		selectedContent = contents.OriginalContent
	}
	lineCount := capturedLineCount(selectedContent)
	if normalisedRequest.EndLine > lineCount {
		return Annotation{}, InvalidAnnotationError{
			Detail: fmt.Sprintf("endLine must not exceed the selected side's %d captured lines", lineCount),
		}
	}
	if err := ctx.Err(); err != nil {
		return Annotation{}, err
	}

	annotationID, err := newAnnotationID()
	if err != nil {
		return Annotation{}, err
	}

	model.mu.Lock()
	defer model.mu.Unlock()
	currentRecord, found := model.items[snapshotID]
	if !found || currentRecord != record {
		return Annotation{}, SnapshotNotFoundError{SnapshotID: snapshotID}
	}
	if len(record.annotations) >= maximumAnnotationsPerSnapshot {
		return Annotation{}, InvalidAnnotationError{Detail: "a review snapshot may contain at most 500 annotations"}
	}

	annotation := Annotation{
		ID:         annotationID,
		SnapshotID: snapshotID,
		FileID:     normalisedRequest.FileID,
		Scope:      normalisedRequest.Scope,
		Side:       normalisedRequest.Side,
		StartLine:  normalisedRequest.StartLine,
		EndLine:    normalisedRequest.EndLine,
		Summary:    normalisedRequest.Summary,
		Rationale:  cloneString(normalisedRequest.Rationale),
		Author:     cloneString(normalisedRequest.Author),
		CreatedAt:  time.Now().UTC(),
	}
	record.annotations = append(record.annotations, annotation)
	record.annotationRevision++
	publishAnnotationRevision(record)
	return cloneAnnotation(annotation), nil
}

// ListAnnotations returns detached annotations in creation order.
func (model *Model) ListAnnotations(snapshotID string) ([]Annotation, error) {
	model.mu.RLock()
	defer model.mu.RUnlock()

	record, found := model.items[snapshotID]
	if !found {
		return nil, SnapshotNotFoundError{SnapshotID: snapshotID}
	}

	annotations := make([]Annotation, len(record.annotations))
	for index, annotation := range record.annotations {
		annotations[index] = cloneAnnotation(annotation)
	}
	return annotations, nil
}

// DeleteAnnotation removes one annotation and publishes the new collection revision.
func (model *Model) DeleteAnnotation(snapshotID string, annotationID string) error {
	model.mu.Lock()
	defer model.mu.Unlock()

	record, found := model.items[snapshotID]
	if !found {
		return SnapshotNotFoundError{SnapshotID: snapshotID}
	}

	annotationIndex := -1
	for index, annotation := range record.annotations {
		if annotation.ID == annotationID {
			annotationIndex = index
			break
		}
	}
	if annotationIndex < 0 {
		return AnnotationNotFoundError{SnapshotID: snapshotID, AnnotationID: annotationID}
	}

	lastAnnotationIndex := len(record.annotations) - 1
	copy(record.annotations[annotationIndex:], record.annotations[annotationIndex+1:])
	record.annotations[lastAnnotationIndex] = Annotation{}
	record.annotations = record.annotations[:lastAnnotationIndex]
	record.annotationRevision++
	publishAnnotationRevision(record)
	return nil
}

// SubscribeAnnotations opens a coalescing revision stream and immediately publishes the current revision.
func (model *Model) SubscribeAnnotations(snapshotID string) (AnnotationSubscription, error) {
	model.mu.Lock()
	defer model.mu.Unlock()

	record, found := model.items[snapshotID]
	if !found {
		return nil, SnapshotNotFoundError{SnapshotID: snapshotID}
	}

	record.nextSubscriberID++
	subscriberID := record.nextSubscriberID
	updates := make(chan AnnotationRevision, 1)
	updates <- AnnotationRevision{Revision: record.annotationRevision}
	record.annotationSubscribers[subscriberID] = updates

	return &annotationSubscription{
		updates: updates,
		close: func() {
			model.mu.Lock()
			defer model.mu.Unlock()
			currentRecord, found := model.items[snapshotID]
			if !found || currentRecord.annotationSubscribers[subscriberID] != updates {
				return
			}
			delete(currentRecord.annotationSubscribers, subscriberID)
			close(updates)
		},
	}, nil
}

// Updates returns the subscription's revision channel.
func (subscription *annotationSubscription) Updates() <-chan AnnotationRevision {
	return subscription.updates
}

// Close releases the subscription and closes its update channel.
func (subscription *annotationSubscription) Close() {
	subscription.once.Do(subscription.close)
}

func normaliseAnnotationRequest(request CreateAnnotationRequest) (CreateAnnotationRequest, error) {
	request.Summary = strings.TrimSpace(request.Summary)
	request.Rationale = trimOptionalAnnotationText(request.Rationale)
	request.Author = trimOptionalAnnotationText(request.Author)

	switch {
	case request.FileID == "":
		return CreateAnnotationRequest{}, InvalidAnnotationError{Detail: "fileId is required"}
	case request.Scope != ScopeWorkingTree && request.Scope != ScopeLastCommit && request.Scope != ScopePullRequest:
		return CreateAnnotationRequest{}, InvalidAnnotationError{Detail: "scope must be one of working-tree, last-commit, or pull-request"}
	case request.Side != AnnotationSideOriginal && request.Side != AnnotationSideModified:
		return CreateAnnotationRequest{}, InvalidAnnotationError{Detail: "side must be original or modified"}
	case request.StartLine < 1:
		return CreateAnnotationRequest{}, InvalidAnnotationError{Detail: "startLine must be a positive one-based line number"}
	case request.EndLine < 1:
		return CreateAnnotationRequest{}, InvalidAnnotationError{Detail: "endLine must be a positive one-based line number"}
	case request.EndLine < request.StartLine:
		return CreateAnnotationRequest{}, InvalidAnnotationError{Detail: "endLine must be greater than or equal to startLine"}
	case request.Summary == "":
		return CreateAnnotationRequest{}, InvalidAnnotationError{Detail: "summary is required"}
	case len(request.Summary) > maximumAnnotationSummaryBytes:
		return CreateAnnotationRequest{}, InvalidAnnotationError{Detail: "summary must contain at most 4096 UTF-8 bytes"}
	case request.Rationale != nil && len(*request.Rationale) > maximumAnnotationRationaleBytes:
		return CreateAnnotationRequest{}, InvalidAnnotationError{Detail: "rationale must contain at most 16384 UTF-8 bytes"}
	case request.Author != nil && len(*request.Author) > maximumAnnotationAuthorBytes:
		return CreateAnnotationRequest{}, InvalidAnnotationError{Detail: "author must contain at most 128 UTF-8 bytes"}
	default:
		return request, nil
	}
}

func trimOptionalAnnotationText(value *string) *string {
	if value == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func capturedLineCount(content string) int {
	if content == "" {
		return 0
	}

	lineCount := strings.Count(content, "\n")
	if !strings.HasSuffix(content, "\n") {
		lineCount++
	}
	return lineCount
}

func publishAnnotationRevision(record *snapshotRecord) {
	revision := AnnotationRevision{Revision: record.annotationRevision}
	for _, updates := range record.annotationSubscribers {
		select {
		case updates <- revision:
			continue
		default:
		}

		select {
		case <-updates:
		default:
		}
		select {
		case updates <- revision:
		default:
		}
	}
}

func cloneAnnotation(annotation Annotation) Annotation {
	cloned := annotation
	cloned.Rationale = cloneString(annotation.Rationale)
	cloned.Author = cloneString(annotation.Author)
	return cloned
}

func newAnnotationID() (string, error) {
	bytes := make([]byte, 12)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generating review annotation ID: %w", err)
	}
	return "review_annotation_" + hex.EncodeToString(bytes), nil
}
