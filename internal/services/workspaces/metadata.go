package workspaces

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

type workspaceMetadata struct {
	branch             *Branch
	remoteRepositoryID *string
	links              WorkspaceLinks
}

func loadMetadata(ctx context.Context, workspacePath string, git GitRepository, github GitHubRepository) workspaceMetadata {
	branchName := currentGitBranch(ctx, workspacePath, git)
	remoteRepositoryID := githubRepositoryID(ctx, workspacePath, git)

	var branch *Branch
	if branchName != "" {
		branch = &Branch{
			Ref:  "refs/heads/" + branchName,
			Name: branchName,
		}
	}

	var remoteRepositoryIDReference *string
	if remoteRepositoryID != "" {
		remoteRepositoryIDReference = &remoteRepositoryID
	}

	return workspaceMetadata{
		branch:             branch,
		remoteRepositoryID: remoteRepositoryIDReference,
		links: WorkspaceLinks{
			Repository:  stringReference(repositoryURL(remoteRepositoryID)),
			PullRequest: stringReference(pullRequestURL(ctx, remoteRepositoryID, branchName, github)),
			Issue:       issueReference(branchName),
		},
	}
}

func currentGitBranch(ctx context.Context, workspacePath string, git GitRepository) string {
	if git == nil {
		return ""
	}

	output, err := git.CurrentBranch(ctx, workspacePath)
	if err != nil {
		return ""
	}

	return output
}

func issueReference(branchName string) *IssueReference {
	matches := linearTicketPattern.FindStringSubmatch(branchName)
	if len(matches) < 2 {
		return nil
	}

	key := strings.ToUpper(matches[1])
	return &IssueReference{
		Provider: "linear",
		Key:      key,
		URL:      fmt.Sprintf("https://linear.app/%s/issue/%s", linearWorkspace, key),
	}
}

func pullRequestURL(ctx context.Context, remoteRepositoryID string, branchName string, github GitHubRepository) string {
	if branchName == "" || remoteRepositoryID == "" || github == nil {
		return ""
	}

	url, err := github.PullRequestURL(ctx, remoteRepositoryID, branchName)
	if err != nil {
		return ""
	}

	return url
}

func githubRepositoryID(ctx context.Context, workspacePath string, git GitRepository) string {
	if git == nil {
		return ""
	}

	remoteURL, err := git.OriginURL(ctx, workspacePath)
	if err != nil {
		return ""
	}

	return parseGitHubRepositoryID(remoteURL)
}

func repositoryURL(remoteRepositoryID string) string {
	if remoteRepositoryID == "" {
		return ""
	}

	if strings.Count(remoteRepositoryID, "/") == 1 {
		return fmt.Sprintf("https://github.com/%s", remoteRepositoryID)
	}

	return fmt.Sprintf("https://%s", remoteRepositoryID)
}

func parseGitHubRepositoryID(remoteURL string) string {
	trimmedRemoteURL := strings.TrimSuffix(remoteURL, ".git")
	for _, pattern := range gitRemotePatterns {
		matches := pattern.FindStringSubmatch(trimmedRemoteURL)
		if len(matches) < 4 {
			continue
		}

		host := matches[1]
		owner := matches[2]
		repository := matches[3]

		if host == "github.com" {
			return fmt.Sprintf("%s/%s", owner, repository)
		}

		return fmt.Sprintf("%s/%s/%s", host, owner, repository)
	}

	return ""
}

func stringReference(value string) *string {
	if value == "" {
		return nil
	}

	return &value
}
