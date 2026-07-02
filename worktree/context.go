// NOTE: Vibecoded and not suppppppper reviewed
package worktree

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

type contextData struct {
	mainPath    string
	projectName string
	worktrees   []porcelainWorktree
}

type porcelainWorktree struct {
	path   string
	branch string
}

func resolveContext(ctx context.Context, projectPath string) (contextData, error) {
	entries, err := listWorktreeEntries(ctx, projectPath)
	if err != nil {
		return contextData{}, err
	}

	if len(entries) == 0 {
		return contextData{}, errors.New("could not determine main worktree path")
	}

	mainPath := entries[0].path
	return contextData{
		mainPath:    mainPath,
		projectName: filepath.Base(mainPath),
		worktrees:   entries,
	}, nil
}

func listWorktreeEntries(ctx context.Context, projectPath string) ([]porcelainWorktree, error) {
	output, err := gitOutput(ctx, projectPath, "worktree", "list", "--porcelain")
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
			current.branch = strings.TrimPrefix(branchRef, "refs/heads/")
			continue
		}
	}

	if current != nil {
		entries = append(entries, *current)
	}

	return entries, nil
}

func buildWorktrees(data contextData, currentPath string) []Worktree {
	worktrees := make([]Worktree, 0, len(data.worktrees))
	prefix := data.projectName + "-"

	for _, entry := range data.worktrees {
		projectName := filepath.Base(entry.path)
		name := projectName
		isBase := samePath(entry.path, data.mainPath)
		if isBase {
			name = data.projectName
		} else {
			name = strings.TrimPrefix(projectName, prefix)
		}

		worktrees = append(worktrees, Worktree{
			Name:        name,
			ProjectName: projectName,
			Path:        entry.path,
			Branch:      entry.branch,
			IsBase:      isBase,
			IsCurrent:   samePath(entry.path, currentPath),
			IsRemovable: !isBase,
		})
	}

	return worktrees
}

func worktreesByBranch(worktrees []Worktree) map[string]Worktree {
	result := map[string]Worktree{}
	for _, worktree := range worktrees {
		if worktree.Branch == "" {
			continue
		}
		result[worktree.Branch] = worktree
	}
	return result
}
