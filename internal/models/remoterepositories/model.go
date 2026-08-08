package remoterepositories

import (
	"context"
	"errors"
	"regexp"
	"sort"
	"strings"

	"wade/internal/infrastructure/github"
)

var remoteRepositoryIDPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)

// Model owns provider repository mapping and validation.
type Model struct {
	github GitHub
}

// New constructs a focused RemoteRepositories Model.
func New(github GitHub) *Model {
	return &Model{github: github}
}

// List returns validated and sorted detached remote repository snapshots.
func (model *Model) List(ctx context.Context) ([]RemoteRepository, error) {
	repositories, err := model.github.ListRepositories(ctx)
	if err != nil {
		return nil, err
	}

	remoteRepositories := make([]RemoteRepository, 0, len(repositories))
	for _, repository := range repositories {
		remoteRepository, err := buildRemoteRepository(repository)
		if err != nil {
			return nil, err
		}
		remoteRepositories = append(remoteRepositories, remoteRepository)
	}
	sort.Slice(remoteRepositories, func(firstIndex int, secondIndex int) bool {
		return remoteRepositories[firstIndex].ID < remoteRepositories[secondIndex].ID
	})
	return remoteRepositories, nil
}

func buildRemoteRepository(repository github.Repository) (RemoteRepository, error) {
	name := strings.TrimSpace(repository.Name)
	remoteRepositoryID := strings.TrimSpace(repository.NameWithOwner)
	webURL := strings.TrimSpace(repository.URL)
	cloneURL := strings.TrimSpace(repository.SSHURL)
	if name == "" || remoteRepositoryID == "" || webURL == "" || cloneURL == "" {
		return RemoteRepository{}, errors.New("GitHub repository response is missing required fields")
	}
	if !remoteRepositoryIDPattern.MatchString(remoteRepositoryID) {
		return RemoteRepository{}, InvalidRemoteRepositoryIDError{RemoteRepositoryID: remoteRepositoryID}
	}

	return RemoteRepository{
		ID:                remoteRepositoryID,
		Name:              name,
		WebURL:            webURL,
		CloneURL:          cloneURL,
		LocalWorkspaceIDs: []string{},
	}, nil
}
