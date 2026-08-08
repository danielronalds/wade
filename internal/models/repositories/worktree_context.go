package repositories

import (
	"context"
	"fmt"

	"wade/internal/infrastructure/git"
)

func listWorktreeEntries(ctx context.Context, repositoryPath string, git Git) ([]git.Worktree, error) {
	worktrees, err := git.Worktrees(ctx, repositoryPath)
	if err != nil {
		return nil, fmt.Errorf("listing worktrees: %w", err)
	}
	return worktrees, nil
}
