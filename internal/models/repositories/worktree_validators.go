package repositories

import (
	"fmt"
	"path/filepath"
)

func validateWorktreeID(worktreeID string) error {
	if worktreeID == "" || worktreeID == "." || worktreeID == ".." {
		return InvalidWorktreeIDError{WorktreeID: worktreeID}
	}
	if filepath.IsAbs(worktreeID) || filepath.Base(worktreeID) != worktreeID {
		return InvalidWorktreeIDError{WorktreeID: worktreeID}
	}

	return nil
}

func validateBranchKind(kind BranchKind) error {
	if kind != BranchKindLocal && kind != BranchKindRemote {
		return fmt.Errorf("invalid branch kind %q", kind)
	}

	return nil
}
