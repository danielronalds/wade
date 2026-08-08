package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const commandTimeout = 2 * time.Minute

// CommandRunner runs a provider command and returns its standard output.
type CommandRunner func(ctx context.Context, name string, args ...string) (string, error)

// Repository is the technical GitHub repository representation.
type Repository struct {
	Name          string `json:"name"`
	NameWithOwner string `json:"nameWithOwner"`
	URL           string `json:"url"`
	SSHURL        string `json:"sshUrl"`
}

// PullRequest is parsed technical pull request metadata.
type PullRequest struct {
	Number      int    `json:"number"`
	URL         string `json:"url"`
	State       string `json:"state"`
	BaseRefName string `json:"baseRefName"`
	HeadRefName string `json:"headRefName"`
}

// Client executes GitHub provider operations.
type Client struct {
	runner          CommandRunner
	directoryRunner func(ctx context.Context, directory string, name string, args ...string) (string, error)
}

// NewClient constructs a GitHub client using the supplied command runner.
func NewClient(runner CommandRunner) Client {
	return Client{runner: runner, directoryRunner: runCommandInDirectory}
}

// ListRepositories returns parsed repositories visible to the current GitHub account.
func (client Client) ListRepositories(ctx context.Context) ([]Repository, error) {
	output, err := client.run(ctx, "repo", "list", "--json", "name,nameWithOwner,url,sshUrl", "--limit", "5000")
	if err != nil {
		return nil, fmt.Errorf("listing GitHub repositories: %w", err)
	}

	var repositories []Repository
	if err := json.Unmarshal([]byte(output), &repositories); err != nil {
		return nil, fmt.Errorf("parsing GitHub repositories: %w", err)
	}
	return repositories, nil
}

// CloneRepository clones a provider repository into targetPath.
func (client Client) CloneRepository(ctx context.Context, nameWithOwner string, targetPath string) error {
	if _, err := client.run(ctx, "repo", "clone", nameWithOwner, targetPath); err != nil {
		return fmt.Errorf("cloning GitHub repository: %w", err)
	}
	return nil
}

// PullRequestURL resolves a pull request URL for a branch.
func (client Client) PullRequestURL(ctx context.Context, repository string, branch string) (string, error) {
	url, err := client.run(ctx, "pr", "view", branch, "--repo", repository, "--json", "url", "--jq", ".url")
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(url) == "" {
		return "", nil
	}
	return url, nil
}

// PullRequest returns parsed pull request metadata for a local repository branch.
func (client Client) PullRequest(ctx context.Context, repoRoot string, branch string) (*PullRequest, error) {
	output, err := client.runInDirectory(ctx, repoRoot, "pr", "view", branch, "--json", "number,url,state,baseRefName,headRefName")
	if err != nil {
		return nil, err
	}

	var pullRequest PullRequest
	if err := json.Unmarshal([]byte(output), &pullRequest); err != nil {
		return nil, fmt.Errorf("parsing GitHub pull request: %w", err)
	}
	return &pullRequest, nil
}

// RunCommand executes a command in the current process directory.
func RunCommand(ctx context.Context, name string, args ...string) (string, error) {
	return runCommandInDirectory(ctx, "", name, args...)
}

func (client Client) run(ctx context.Context, args ...string) (string, error) {
	if client.runner == nil {
		return "", errors.New("command runner is required")
	}

	commandContext, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()
	return client.runner(commandContext, "gh", args...)
}

func (client Client) runInDirectory(ctx context.Context, directory string, args ...string) (string, error) {
	if client.directoryRunner == nil {
		return "", errors.New("directory command runner is required")
	}

	commandContext, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()
	return client.directoryRunner(commandContext, directory, "gh", args...)
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
		return "", errors.New(text)
	}
	return text, nil
}
