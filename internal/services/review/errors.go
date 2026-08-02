package review

import "fmt"

type SnapshotNotFoundError struct {
	SnapshotID string
}

func (e SnapshotNotFoundError) Error() string {
	return fmt.Sprintf("review snapshot %q not found", e.SnapshotID)
}

type SnapshotFileNotFoundError struct {
	SnapshotID string
	FileID     string
}

func (e SnapshotFileNotFoundError) Error() string {
	return fmt.Sprintf("file %q was not found in review snapshot %q", e.FileID, e.SnapshotID)
}
