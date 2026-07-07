// NOTE: Vibecoded and not suppppppper reviewed
package worktree

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
)

func preferredRemote(ctx context.Context, projectPath string) (string, bool, error) {
	output, err := gitOutput(ctx, projectPath, "remote")
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

func fetchRemote(ctx context.Context, projectPath string, remote string) error {
	if _, err := gitOutput(ctx, projectPath, "fetch", remote, "--prune"); err != nil {
		return fmt.Errorf("fetching remote %q: %w", remote, err)
	}
	return nil
}

func listRemoteBranches(ctx context.Context, projectPath string, remote string) ([]string, error) {
	output, err := gitOutput(ctx, projectPath, "branch", "-r", "--format=%(refname:short)")
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

func listLocalBranches(ctx context.Context, projectPath string) (map[string]bool, error) {
	output, err := gitOutput(ctx, projectPath, "branch", "--format=%(refname:short)")
	if err != nil {
		return nil, fmt.Errorf("listing local branches: %w", err)
	}

	branches := map[string]bool{}
	for _, branch := range parseLines(output) {
		branches[branch] = true
	}
	return branches, nil
}

func validateBranchName(ctx context.Context, projectPath string, branch string) error {
	if branch == "" {
		return errors.New("branch is required")
	}

	if _, err := gitOutput(ctx, projectPath, "check-ref-format", "--branch", branch); err != nil {
		return fmt.Errorf("invalid branch %q", branch)
	}

	return nil
}
