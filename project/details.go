package project

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

const linearWorkspace = "signinsolutions"

var (
	linearTicketPattern = regexp.MustCompile(`([a-zA-Z]+-[0-9]+)`)
	gitRemotePatterns   = []*regexp.Regexp{
		regexp.MustCompile(`^git@([^:]+):([^/]+)/(.+)$`),
		regexp.MustCompile(`^https?://([^/]+)/([^/]+)/(.+)$`),
		regexp.MustCompile(`^ssh://git@([^/]+)/([^/]+)/(.+)$`),
	}
)

type Metadata struct {
	GitBranch       string
	LinearTicketURL string
	PullRequestURL  string
}

func Details(projectPath string) Metadata {
	gitBranch := currentGitBranch(projectPath)

	return Metadata{
		GitBranch:       gitBranch,
		LinearTicketURL: linearTicketURL(gitBranch),
		PullRequestURL:  pullRequestURL(projectPath, gitBranch),
	}
}

func currentGitBranch(projectPath string) string {
	output, err := commandOutput(time.Second, "git", "-C", projectPath, "branch", "--show-current")
	if err != nil {
		return ""
	}

	return output
}

func linearTicketURL(gitBranch string) string {
	matches := linearTicketPattern.FindStringSubmatch(gitBranch)
	if len(matches) < 2 {
		return ""
	}

	return fmt.Sprintf("https://linear.app/%s/issue/%s", linearWorkspace, strings.ToUpper(matches[1]))
}

func pullRequestURL(projectPath string, gitBranch string) string {
	if gitBranch == "" {
		return ""
	}

	repo := githubRepo(projectPath)
	if repo == "" {
		return ""
	}

	url, err := commandOutput(
		3*time.Second,
		"gh",
		"pr",
		"view",
		gitBranch,
		"--repo",
		repo,
		"--json",
		"url",
		"--jq",
		".url",
	)
	if err != nil {
		return ""
	}

	return url
}

func githubRepo(projectPath string) string {
	remoteURL, err := commandOutput(time.Second, "git", "-C", projectPath, "remote", "get-url", "origin")
	if err != nil {
		return ""
	}

	return parseGitHubRepo(remoteURL)
}

func parseGitHubRepo(remoteURL string) string {
	trimmedRemoteURL := strings.TrimSuffix(remoteURL, ".git")
	for _, pattern := range gitRemotePatterns {
		matches := pattern.FindStringSubmatch(trimmedRemoteURL)
		if len(matches) < 4 {
			continue
		}

		host := matches[1]
		owner := matches[2]
		repo := matches[3]

		if host == "github.com" {
			return fmt.Sprintf("%s/%s", owner, repo)
		}

		return fmt.Sprintf("%s/%s/%s", host, owner, repo)
	}

	return ""
}

func commandOutput(timeout time.Duration, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	output, err := exec.CommandContext(ctx, name, args...).Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}
