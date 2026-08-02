package worktrees

import "fmt"

type InvalidWorktreeIDError struct {
	WorktreeID string
}

func (e InvalidWorktreeIDError) Error() string {
	return fmt.Sprintf("invalid worktree ID %q", e.WorktreeID)
}

type WorktreeNotFoundError struct {
	WorktreeID string
}

func (e WorktreeNotFoundError) Error() string {
	return fmt.Sprintf("worktree %q not found", e.WorktreeID)
}

type WorktreeNotRemovableError struct {
	WorktreeID string
}

func (e WorktreeNotRemovableError) Error() string {
	return fmt.Sprintf("worktree %q cannot be removed", e.WorktreeID)
}
