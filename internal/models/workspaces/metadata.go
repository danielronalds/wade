package workspaces

import "fmt"

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
