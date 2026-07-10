package projects

// TODO: Review properly

import (
	"context"
	"fmt"
	"regexp"
	"strings"
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
	GitHubURL       string
}

func Details(ctx context.Context, projectPath string, git GitRepository, github GitHubRepository) Metadata {
	gitBranch := currentGitBranch(ctx, projectPath, git)
	repo := githubRepo(ctx, projectPath, git)

	return Metadata{
		GitBranch:       gitBranch,
		LinearTicketURL: linearTicketURL(gitBranch),
		PullRequestURL:  pullRequestURL(ctx, repo, gitBranch, github),
		GitHubURL:       githubURL(repo),
	}
}

func currentGitBranch(ctx context.Context, projectPath string, git GitRepository) string {
	if git == nil {
		return ""
	}

	output, err := git.CurrentBranch(ctx, projectPath)
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

func pullRequestURL(ctx context.Context, repo string, gitBranch string, github GitHubRepository) string {
	if gitBranch == "" || repo == "" || github == nil {
		return ""
	}

	url, err := github.PullRequestURL(ctx, repo, gitBranch)
	if err != nil {
		return ""
	}

	return url
}

func githubRepo(ctx context.Context, projectPath string, git GitRepository) string {
	if git == nil {
		return ""
	}

	remoteURL, err := git.OriginURL(ctx, projectPath)
	if err != nil {
		return ""
	}

	return parseGitHubRepo(remoteURL)
}

func githubURL(repo string) string {
	if repo == "" {
		return ""
	}

	if strings.Count(repo, "/") == 1 {
		return fmt.Sprintf("https://github.com/%s", repo)
	}

	return fmt.Sprintf("https://%s", repo)
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
