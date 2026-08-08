package repositories

import "fmt"

// BranchReferenceRequiredError reports a missing branch reference.
type BranchReferenceRequiredError struct{}

func (BranchReferenceRequiredError) Error() string {
	return "branch reference is required"
}

// InvalidBranchReferenceError reports an unsupported or malformed branch reference.
type InvalidBranchReferenceError struct {
	BranchRef string
}

func (e InvalidBranchReferenceError) Error() string {
	return fmt.Sprintf("invalid branch reference %q", e.BranchRef)
}

// InvalidWorktreeIDError reports a malformed worktree identity.
type InvalidWorktreeIDError struct {
	WorktreeID string
}

func (e InvalidWorktreeIDError) Error() string {
	return fmt.Sprintf("invalid worktree ID %q", e.WorktreeID)
}

// WorktreeNotFoundError reports an unknown worktree identity.
type WorktreeNotFoundError struct {
	WorktreeID string
}

func (e WorktreeNotFoundError) Error() string {
	return fmt.Sprintf("worktree %q not found", e.WorktreeID)
}

// WorktreeNotRemovableError reports a worktree protected from removal.
type WorktreeNotRemovableError struct {
	WorktreeID string
}

func (e WorktreeNotRemovableError) Error() string {
	return fmt.Sprintf("worktree %q cannot be removed", e.WorktreeID)
}
