package workspaces

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"wade/internal/services/gitrepositories"
)

type workspaceRepositoryStub struct {
	workspaceIDs []string
	path         string
	found        bool
	err          error
}

func (s workspaceRepositoryStub) IDs() ([]string, error) {
	return s.workspaceIDs, s.err
}

func (s workspaceRepositoryStub) Resolve(string) (string, bool, error) {
	return s.path, s.found, s.err
}

func (workspaceRepositoryStub) IDForDirectory(string) (string, bool, error) {
	return "", false, nil
}

func (workspaceRepositoryStub) IDsForDirectories([]string) ([]string, error) {
	return nil, nil
}

func (workspaceRepositoryStub) Directories() []string {
	return nil
}

func (workspaceRepositoryStub) Reload([]string) {}

type localRepositoryServiceStub struct {
	contexts         []gitrepositories.WorkspaceContext
	workspaceContext gitrepositories.WorkspaceContext
	isGit            bool
	err              error
}

func (s localRepositoryServiceStub) ListWorkspaceContexts(context.Context) ([]gitrepositories.WorkspaceContext, error) {
	return s.contexts, s.err
}

func (s localRepositoryServiceStub) ResolveWorkspace(context.Context, string) (gitrepositories.WorkspaceContext, bool, error) {
	return s.workspaceContext, s.isGit, s.err
}

type terminalActivityStub map[string]int

func (s terminalActivityStub) ActiveTerminalCount(workspaceID string) int {
	return s[workspaceID]
}

type workspaceGitHubRepositoryStub struct {
	pullRequestURL string
}

func (s workspaceGitHubRepositoryStub) PullRequestURL(context.Context, string, string) (string, error) {
	return s.pullRequestURL, nil
}

func TestServiceListReturnsWorkspaceSummaries(t *testing.T) {
	service := NewService(
		workspaceRepositoryStub{workspaceIDs: []string{"alpha", "bravo"}},
		localRepositoryServiceStub{},
		nil,
	)
	service.SetTerminalActivity(terminalActivityStub{"alpha": 2})

	got, err := service.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v, want nil", err)
	}

	want := []WorkspaceSummary{
		{ID: "alpha", Name: "alpha", Activity: WorkspaceActivity{ActiveTerminalCount: 2}},
		{ID: "bravo", Name: "bravo"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("List() = %#v, want %#v", got, want)
	}
}

func TestServiceGetReturnsWorkspaceRepositoryMetadata(t *testing.T) {
	remoteRepositoryID := "example/wade"
	localContext := gitrepositories.WorkspaceContext{
		RepositoryContext: gitrepositories.Context{Repository: gitrepositories.Repository{
			ID:                 "wade",
			RemoteRepositoryID: &remoteRepositoryID,
			MainWorkspaceID:    "wade",
			WorkspaceIDs:       []string{"wade"},
		}},
		WorkspaceID: "wade",
		Branch: gitrepositories.Branch{
			Ref:    "refs/heads/feature/wade-123-workspaces",
			Name:   "feature/wade-123-workspaces",
			Commit: "abc123",
		},
		IsMain:      true,
		IsRemovable: false,
	}
	service := NewService(
		workspaceRepositoryStub{path: "/workspaces/wade", found: true},
		localRepositoryServiceStub{workspaceContext: localContext, isGit: true},
		workspaceGitHubRepositoryStub{pullRequestURL: "https://github.com/example/wade/pull/123"},
	)

	got, err := service.Get(context.Background(), "wade")
	if err != nil {
		t.Fatalf("Get() error = %v, want nil", err)
	}

	if got.ID != "wade" || got.Name != "wade" {
		t.Fatalf("Get() identity = %q/%q, want wade/wade", got.ID, got.Name)
	}
	if got.RepositoryID == nil || *got.RepositoryID != "wade" {
		t.Fatalf("Get() RepositoryID = %#v, want wade", got.RepositoryID)
	}
	if got.RemoteRepositoryID == nil || *got.RemoteRepositoryID != "example/wade" {
		t.Fatalf("Get() RemoteRepositoryID = %#v, want example/wade", got.RemoteRepositoryID)
	}
	if got.Worktree == nil || !got.Worktree.IsMain || got.Worktree.IsRemovable {
		t.Fatalf("Get() Worktree = %#v, want non-removable main worktree", got.Worktree)
	}
	if got.Branch == nil || got.Branch.Ref != "refs/heads/feature/wade-123-workspaces" || got.Branch.Commit != "abc123" {
		t.Fatalf("Get() Branch = %#v, want feature branch at abc123", got.Branch)
	}
	if got.Links.Repository == nil || *got.Links.Repository != "https://github.com/example/wade" {
		t.Fatalf("Get() repository link = %#v, want GitHub URL", got.Links.Repository)
	}
	if got.Links.PullRequest == nil || *got.Links.PullRequest != "https://github.com/example/wade/pull/123" {
		t.Fatalf("Get() pull request link = %#v, want pull request URL", got.Links.PullRequest)
	}
	if got.Links.Issue == nil || got.Links.Issue.Key != "WADE-123" {
		t.Fatalf("Get() issue link = %#v, want WADE-123", got.Links.Issue)
	}
}

func TestServiceGetReturnsNullGitRelationshipsForNonGitWorkspace(t *testing.T) {
	service := NewService(
		workspaceRepositoryStub{path: "/workspaces/notes", found: true},
		localRepositoryServiceStub{},
		nil,
	)

	got, err := service.Get(context.Background(), "notes")
	if err != nil {
		t.Fatalf("Get() error = %v, want nil", err)
	}
	if got.RepositoryID != nil || got.RemoteRepositoryID != nil || got.Worktree != nil || got.Branch != nil {
		t.Fatalf("Get() Git relationships = %#v, want nil", got)
	}
}

func TestServicePathReturnsTypedNotFoundError(t *testing.T) {
	service := NewService(workspaceRepositoryStub{}, localRepositoryServiceStub{}, nil)

	_, err := service.Path("missing")

	var notFoundError WorkspaceNotFoundError
	if !errors.As(err, &notFoundError) {
		t.Fatalf("Path() error = %v, want WorkspaceNotFoundError", err)
	}
}

func TestServicePathReturnsTypedInvalidIDError(t *testing.T) {
	service := NewService(workspaceRepositoryStub{}, localRepositoryServiceStub{}, nil)

	_, err := service.Path("../wade")

	var invalidIDError InvalidWorkspaceIDError
	if !errors.As(err, &invalidIDError) {
		t.Fatalf("Path() error = %v, want InvalidWorkspaceIDError", err)
	}
}
