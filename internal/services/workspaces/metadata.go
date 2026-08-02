package workspaces

// TODO: Review properly

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

const linearWorkspace = "signinsolutions"

var linearTicketPattern = regexp.MustCompile(`([a-zA-Z]+-[0-9]+)`)

func workspaceLinks(
	ctx context.Context,
	remoteRepositoryID *string,
	branchName string,
	github GitHubRepository,
) WorkspaceLinks {
	if remoteRepositoryID == nil {
		return WorkspaceLinks{Issue: issueReference(branchName)}
	}

	return WorkspaceLinks{
		Repository:  stringReference(repositoryURL(*remoteRepositoryID)),
		PullRequest: stringReference(pullRequestURL(ctx, *remoteRepositoryID, branchName, github)),
		Issue:       issueReference(branchName),
	}
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

func repositoryURL(remoteRepositoryID string) string {
	if remoteRepositoryID == "" {
		return ""
	}

	return fmt.Sprintf("https://github.com/%s", remoteRepositoryID)
}

func stringReference(value string) *string {
	if value == "" {
		return nil
	}

	return &value
}
