// NOTE: Vibecoded and not suppppppper reviewed
package worktrees

// TODO: Review properly

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
)

func preferredRemote(ctx context.Context, projectPath string, git gitRepository) (string, bool, error) {
	output, err := git.Remotes(ctx, projectPath)
	if err != nil {
		return "", false, fmt.Errorf("listing remotes: %w", err)
	}

	remotes := parseLines(output)
	if len(remotes) == 0 {
		return "", false, nil
	}

	if slices.Contains(remotes, "origin") {
		return "origin", true, nil
	}

	if len(remotes) == 1 {
		return remotes[0], true, nil
	}

	return "", false, errors.New("multiple git remotes found and none is origin")
}

func fetchRemote(ctx context.Context, projectPath string, remote string, git gitRepository) error {
	if err := git.FetchRemote(ctx, projectPath, remote); err != nil {
		return fmt.Errorf("fetching remote %q: %w", remote, err)
	}
	return nil
}

func listRemoteBranches(ctx context.Context, projectPath string, remote string, git gitRepository) ([]string, error) {
	output, err := git.RemoteBranches(ctx, projectPath)
	if err != nil {
		return nil, fmt.Errorf("listing remote branches: %w", err)
	}

	prefix := remote + "/"
	branches := make([]string, 0)
	for _, line := range parseLines(output) {
		if !strings.HasPrefix(line, prefix) || line == remote+"/HEAD" || strings.HasSuffix(line, "/HEAD") {
			continue
		}
		branches = append(branches, strings.TrimPrefix(line, prefix))
	}

	slices.Sort(branches)
	return branches, nil
}

func listLocalBranches(ctx context.Context, projectPath string, git gitRepository) (map[string]bool, error) {
	output, err := git.LocalBranches(ctx, projectPath)
	if err != nil {
		return nil, fmt.Errorf("listing local branches: %w", err)
	}

	branches := map[string]bool{}
	for _, branch := range parseLines(output) {
		branches[branch] = true
	}
	return branches, nil
}

func validateBranchName(ctx context.Context, projectPath string, branch string, git gitRepository) error {
	if branch == "" {
		return errors.New("branch is required")
	}

	if err := git.ValidateBranchName(ctx, projectPath, branch); err != nil {
		return fmt.Errorf("invalid branch %q", branch)
	}

	return nil
}
