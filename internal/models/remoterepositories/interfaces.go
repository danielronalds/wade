package remoterepositories

import (
	"context"

	"wade/internal/infrastructure/github"
)

// GitHub provides repositories visible to the current account.
type GitHub interface {
	ListRepositories(ctx context.Context) ([]github.Repository, error)
}
