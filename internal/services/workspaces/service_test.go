package workspaces

import (
	"context"
	"errors"
	"reflect"
	"testing"
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

type workspaceGitRepositoryStub struct {
	branch    string
	originURL string
}

func (s workspaceGitRepositoryStub) CurrentBranch(context.Context, string) (string, error) {
	return s.branch, nil
}

func (s workspaceGitRepositoryStub) OriginURL(context.Context, string) (string, error) {
	return s.originURL, nil
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
		nil,
		nil,
	)

	got, err := service.List()
	if err != nil {
		t.Fatalf("List() error = %v, want nil", err)
	}

	want := []WorkspaceSummary{
		{ID: "alpha", Name: "alpha"},
		{ID: "bravo", Name: "bravo"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("List() = %#v, want %#v", got, want)
	}
}

func TestServiceGetReturnsWorkspaceMetadata(t *testing.T) {
	service := NewService(
		workspaceRepositoryStub{path: "/workspaces/wade", found: true},
		workspaceGitRepositoryStub{
			branch:    "feature/wade-123-workspaces",
			originURL: "git@github.com:example/wade.git",
		},
		workspaceGitHubRepositoryStub{pullRequestURL: "https://github.com/example/wade/pull/123"},
	)

	got, err := service.Get(context.Background(), "wade")
	if err != nil {
		t.Fatalf("Get() error = %v, want nil", err)
	}

	if got.ID != "wade" || got.Name != "wade" {
		t.Fatalf("Get() identity = %q/%q, want wade/wade", got.ID, got.Name)
	}
	if got.RemoteRepositoryID == nil || *got.RemoteRepositoryID != "example/wade" {
		t.Fatalf("Get() RemoteRepositoryID = %#v, want example/wade", got.RemoteRepositoryID)
	}
	if got.Branch == nil || got.Branch.Ref != "refs/heads/feature/wade-123-workspaces" {
		t.Fatalf("Get() Branch = %#v, want feature branch", got.Branch)
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

func TestServicePathReturnsTypedNotFoundError(t *testing.T) {
	service := NewService(workspaceRepositoryStub{}, nil, nil)

	_, err := service.Path("missing")

	var notFoundError WorkspaceNotFoundError
	if !errors.As(err, &notFoundError) {
		t.Fatalf("Path() error = %v, want WorkspaceNotFoundError", err)
	}
}

func TestServicePathReturnsTypedInvalidIDError(t *testing.T) {
	service := NewService(workspaceRepositoryStub{}, nil, nil)

	_, err := service.Path("../wade")

	var invalidIDError InvalidWorkspaceIDError
	if !errors.As(err, &invalidIDError) {
		t.Fatalf("Path() error = %v, want InvalidWorkspaceIDError", err)
	}
}
