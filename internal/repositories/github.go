package repositories

// TODO: Review properly

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const githubCommandTimeout = 2 * time.Minute

type CommandRunner func(ctx context.Context, name string, args ...string) (string, error)

type GitHubRepository struct {
	runner CommandRunner
}

func NewGitHubRepository(runner CommandRunner) GitHubRepository {
	return GitHubRepository{runner: runner}
}

func (r GitHubRepository) ListRepositories(ctx context.Context) (string, error) {
	output, err := r.run(ctx, "repo", "list", "--json", "name,nameWithOwner,url,sshUrl", "--limit", "5000")
	if err != nil {
		return "", fmt.Errorf("listing GitHub repositories: %w", err)
	}

	return output, nil
}

func (r GitHubRepository) CloneRepository(ctx context.Context, nameWithOwner string, targetPath string) error {
	if _, err := r.run(ctx, "repo", "clone", nameWithOwner, targetPath); err != nil {
		return fmt.Errorf("cloning GitHub repository: %w", err)
	}

	return nil
}

func (r GitHubRepository) PullRequestURL(ctx context.Context, repo string, branch string) (string, error) {
	return r.run(ctx, "pr", "view", branch, "--repo", repo, "--json", "url", "--jq", ".url")
}

func (r GitHubRepository) PullRequest(ctx context.Context, repoRoot string, branch string) ([]byte, error) {
	output, err := r.runInDirectory(ctx, repoRoot, "pr", "view", branch, "--json", "number,url,state,baseRefName,headRefName")
	return []byte(output), err
}

func RunCommand(ctx context.Context, name string, args ...string) (string, error) {
	return runCommandInDirectory(ctx, "", name, args...)
}

func (r GitHubRepository) run(ctx context.Context, args ...string) (string, error) {
	if r.runner == nil {
		return "", fmt.Errorf("command runner is required")
	}

	commandContext, cancel := context.WithTimeout(ctx, githubCommandTimeout)
	defer cancel()

	return r.runner(commandContext, "gh", args...)
}

func (r GitHubRepository) runInDirectory(ctx context.Context, directory string, args ...string) (string, error) {
	if r.runner == nil {
		return "", fmt.Errorf("command runner is required")
	}

	commandContext, cancel := context.WithTimeout(ctx, githubCommandTimeout)
	defer cancel()

	return runCommandInDirectory(commandContext, directory, "gh", args...)
}

func runCommandInDirectory(ctx context.Context, directory string, name string, args ...string) (string, error) {
	command := exec.CommandContext(ctx, name, args...)
	command.Dir = directory
	output, err := command.CombinedOutput()
	text := strings.TrimSpace(string(output))
	if err != nil {
		if text == "" {
			return "", err
		}
		return "", fmt.Errorf("%s", text)
	}

	return text, nil
}
