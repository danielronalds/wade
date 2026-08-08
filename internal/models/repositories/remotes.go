package repositories

import (
	"net/url"
	"regexp"
	"strings"
)

var scpRemotePattern = regexp.MustCompile(`^(?:[^@]+@)?([^:]+):(.+)$`)

func CanonicalRemoteIdentity(remoteURL string) string {
	host, repositoryPath := parseRemote(remoteURL)
	if host == "" || repositoryPath == "" {
		return ""
	}

	return strings.ToLower(host + "/" + repositoryPath)
}

func githubRepositoryID(remoteURL string) *string {
	host, repositoryPath := parseRemote(remoteURL)
	if !strings.EqualFold(host, "github.com") || strings.Count(repositoryPath, "/") != 1 {
		return nil
	}

	return &repositoryPath
}

func parseRemote(remoteURL string) (string, string) {
	remoteURL = strings.TrimSpace(remoteURL)
	if remoteURL == "" {
		return "", ""
	}

	if matches := scpRemotePattern.FindStringSubmatch(remoteURL); len(matches) == 3 && !strings.Contains(remoteURL, "://") {
		return normaliseRemoteParts(matches[1], matches[2])
	}

	parsedURL, err := url.Parse(remoteURL)
	if err != nil || parsedURL.Hostname() == "" {
		return "", ""
	}

	return normaliseRemoteParts(parsedURL.Hostname(), parsedURL.Path)
}

func normaliseRemoteParts(host string, repositoryPath string) (string, string) {
	host = strings.ToLower(strings.TrimSpace(host))
	repositoryPath = strings.Trim(strings.TrimSpace(repositoryPath), "/")
	repositoryPath = strings.TrimSuffix(repositoryPath, ".git")
	if host == "" || repositoryPath == "" {
		return "", ""
	}

	return host, repositoryPath
}
