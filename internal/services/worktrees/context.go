// NOTE: Vibecoded and not suppppppper reviewed
package worktrees

// TODO: Review properly

import (
	"context"
	"fmt"
	"strings"
)

type porcelainWorktree struct {
	path   string
	branch string
}

func listWorktreeEntries(ctx context.Context, repositoryPath string, git gitRepository) ([]porcelainWorktree, error) {
	output, err := git.WorktreeListPorcelain(ctx, repositoryPath)
	if err != nil {
		return nil, fmt.Errorf("listing worktrees: %w", err)
	}

	var entries []porcelainWorktree
	var current *porcelainWorktree
	for _, line := range strings.Split(output, "\n") {
		if line == "" {
			continue
		}

		if path, ok := strings.CutPrefix(line, "worktree "); ok {
			if current != nil {
				entries = append(entries, *current)
			}
			current = &porcelainWorktree{path: path}
			continue
		}

		if current == nil {
			continue
		}

		if branchRef, ok := strings.CutPrefix(line, "branch "); ok {
			current.branch = branchRef
		}
	}

	if current != nil {
		entries = append(entries, *current)
	}

	return entries, nil
}
