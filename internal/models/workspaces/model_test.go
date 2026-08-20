package workspaces

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"

	"wade/internal/infrastructure/linear"
)

type workspaceDiscoveryStub struct {
	workspaceIDs []string
	paths        map[string]string
	directories  []string
}

func (stub *workspaceDiscoveryStub) IDs() ([]string, error) {
	return append([]string(nil), stub.workspaceIDs...), nil
}

func (stub *workspaceDiscoveryStub) Resolve(workspaceID string) (string, bool, error) {
	path, found := stub.paths[workspaceID]
	return path, found, nil
}

func (stub *workspaceDiscoveryStub) Directories() []string {
	return append([]string(nil), stub.directories...)
}

func (stub *workspaceDiscoveryStub) Reload(directories []string) {
	stub.directories = append([]string(nil), directories...)
}

type workspaceLinearStub struct {
	calls      int
	workspaces []string
}

func (stub *workspaceLinearStub) TicketForBranch(workspace string, branch string) (*linear.Ticket, error) {
	stub.calls++
	stub.workspaces = append(stub.workspaces, workspace)
	return &linear.Ticket{Key: "ABC-123", URL: "https://linear.app/" + workspace + "/issue/ABC-123"}, nil
}

type workspaceFileSystemStub struct {
	existing map[string]bool
}

func (stub workspaceFileSystemStub) EnsureDirectory(path string) error {
	return os.MkdirAll(path, 0o755)
}

func (stub workspaceFileSystemStub) EnsurePathDoesNotExist(path string) error {
	if stub.existing[path] {
		return os.ErrExist
	}
	return nil
}

type workspaceGitHubStub struct {
	mu          sync.Mutex
	cloneCalls  int
	clonedPath  string
	pullRequest string
	discovery   *workspaceDiscoveryStub
}

func (stub *workspaceGitHubStub) CloneRepository(_ context.Context, _ string, targetPath string) error {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	stub.cloneCalls++
	stub.clonedPath = targetPath
	workspaceID := filepath.Base(targetPath)
	stub.discovery.paths[workspaceID] = targetPath
	stub.discovery.workspaceIDs = append(stub.discovery.workspaceIDs, workspaceID)
	return nil
}

func (stub *workspaceGitHubStub) PullRequestURL(context.Context, string, string) (string, error) {
	return stub.pullRequest, nil
}

func TestModelListsAndTargetsDiscoveredWorkspaces(t *testing.T) {
	discovery := &workspaceDiscoveryStub{
		workspaceIDs: []string{"alpha", "bravo"},
		paths:        map[string]string{"alpha": "/alpha", "bravo": "/bravo"},
	}
	model := New(workspaceFileSystemStub{}, discovery, &workspaceGitHubStub{}, nil, Configuration{})

	got, err := model.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	want := []WorkspaceSummary{{ID: "alpha", Name: "alpha"}, {ID: "bravo", Name: "bravo"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("List() = %#v, want %#v", got, want)
	}

	targeted, err := model.ListByIDs(context.Background(), []string{"missing", "alpha", "alpha"})
	if err != nil {
		t.Fatalf("ListByIDs() error = %v", err)
	}
	if !reflect.DeepEqual(targeted, []WorkspaceSummary{{ID: "alpha", Name: "alpha"}}) {
		t.Fatalf("ListByIDs() = %#v", targeted)
	}
}

func TestModelGetValidatesWorkspaceIdentity(t *testing.T) {
	model := New(workspaceFileSystemStub{}, &workspaceDiscoveryStub{paths: map[string]string{}}, &workspaceGitHubStub{}, nil, Configuration{})

	_, err := model.Get(context.Background(), "../wade")
	var invalidID InvalidWorkspaceIDError
	if !errors.As(err, &invalidID) {
		t.Fatalf("Get() error = %v, want InvalidWorkspaceIDError", err)
	}

	_, err = model.Get(context.Background(), "missing")
	var notFound WorkspaceNotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("Get() error = %v, want WorkspaceNotFoundError", err)
	}
}

func TestModelResolveLinksUsesLatestEnabledLinearConfiguration(t *testing.T) {
	linearClient := &workspaceLinearStub{}
	model := New(workspaceFileSystemStub{}, &workspaceDiscoveryStub{}, &workspaceGitHubStub{}, linearClient, Configuration{})

	links, err := model.ResolveLinks(context.Background(), LinkContext{BranchName: "abc-123"})
	if err != nil || links.Issue != nil || linearClient.calls != 0 {
		t.Fatalf("disabled ResolveLinks() = %#v, %v, calls %d", links, err, linearClient.calls)
	}

	model.Configure(Configuration{Linear: LinearConfiguration{Enabled: true, Workspace: "first"}})
	model.Configure(Configuration{Linear: LinearConfiguration{Enabled: true, Workspace: "second"}})
	links, err = model.ResolveLinks(context.Background(), LinkContext{BranchName: "abc-123"})
	if err != nil || links.Issue == nil || links.Issue.URL != "https://linear.app/second/issue/ABC-123" {
		t.Fatalf("enabled ResolveLinks() = %#v, %v", links, err)
	}
}

func TestModelMaterialiseUsesExactConfiguredDirectory(t *testing.T) {
	workspaceDirectory := t.TempDir()
	discovery := &workspaceDiscoveryStub{paths: make(map[string]string)}
	github := &workspaceGitHubStub{discovery: discovery}
	model := New(workspaceFileSystemStub{}, discovery, github, nil, Configuration{WorkspaceDirectories: []WorkspaceDirectory{{Setting: "~/Code", Path: workspaceDirectory}}})

	workspace, err := model.Materialise(context.Background(), MaterialiseRequest{RemoteRepositoryID: "example/wade", WorkspaceDirectory: "~/Code"})
	if err != nil {
		t.Fatalf("Materialise() error = %v", err)
	}
	if workspace.ID != "wade" || github.clonedPath != filepath.Join(workspaceDirectory, "wade") {
		t.Fatalf("workspace/path = %#v/%q", workspace, github.clonedPath)
	}

	_, err = model.Materialise(context.Background(), MaterialiseRequest{RemoteRepositoryID: "example/other", WorkspaceDirectory: workspaceDirectory})
	var notConfigured WorkspaceDirectoryNotConfiguredError
	if !errors.As(err, &notConfigured) {
		t.Fatalf("Materialise() error = %v, want WorkspaceDirectoryNotConfiguredError", err)
	}
}

func TestModelMaterialiseSerialisesConflictingRequests(t *testing.T) {
	workspaceDirectory := t.TempDir()
	discovery := &workspaceDiscoveryStub{paths: make(map[string]string)}
	github := &workspaceGitHubStub{discovery: discovery}
	model := New(workspaceFileSystemStub{}, discovery, github, nil, Configuration{WorkspaceDirectories: []WorkspaceDirectory{{Setting: "~/Code", Path: workspaceDirectory}}})

	var wait sync.WaitGroup
	errorsFound := make(chan error, 2)
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			_, err := model.Materialise(context.Background(), MaterialiseRequest{RemoteRepositoryID: "example/wade", WorkspaceDirectory: "~/Code"})
			errorsFound <- err
		}()
	}
	wait.Wait()
	close(errorsFound)

	successes := 0
	conflicts := 0
	for err := range errorsFound {
		if err == nil {
			successes++
			continue
		}
		var conflict WorkspaceAlreadyExistsError
		if errors.As(err, &conflict) {
			conflicts++
		}
	}
	if successes != 1 || conflicts != 1 || github.cloneCalls != 1 {
		t.Fatalf("successes/conflicts/clones = %d/%d/%d, want 1/1/1", successes, conflicts, github.cloneCalls)
	}
}
