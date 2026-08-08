package remoterepositories

import "fmt"

// InvalidRemoteRepositoryIDError reports malformed provider repository data.
type InvalidRemoteRepositoryIDError struct {
	RemoteRepositoryID string
}

func (err InvalidRemoteRepositoryIDError) Error() string {
	return fmt.Sprintf("invalid remote repository ID %q", err.RemoteRepositoryID)
}
