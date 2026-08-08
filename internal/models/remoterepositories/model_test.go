package remoterepositories

import (
	"context"
	"testing"

	"wade/internal/infrastructure/github"
)

type gitHubStub struct {
	repositories []github.Repository
}

func (stub gitHubStub) ListRepositories(context.Context) ([]github.Repository, error) {
	return append([]github.Repository(nil), stub.repositories...), nil
}

func TestModelListMapsValidatesAndSortsProviderRepositories(t *testing.T) {
	model := New(gitHubStub{repositories: []github.Repository{
		{Name: "zeta", NameWithOwner: "example/zeta", URL: "https://github.com/example/zeta", SSHURL: "git@github.com:example/zeta.git"},
		{Name: "alpha", NameWithOwner: "example/alpha", URL: "https://github.com/example/alpha", SSHURL: "git@github.com:example/alpha.git"},
	}})

	got, err := model.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(got) != 2 || got[0].ID != "example/alpha" || got[1].ID != "example/zeta" {
		t.Fatalf("List() = %#v", got)
	}
	if got[0].LocalWorkspaceIDs == nil || len(got[0].LocalWorkspaceIDs) != 0 {
		t.Fatalf("LocalWorkspaceIDs = %#v, want empty detached collection", got[0].LocalWorkspaceIDs)
	}
}
