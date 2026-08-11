package repositories

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"wade/internal/infrastructure/filesystem"
	"wade/internal/infrastructure/git"
)

func TestGetWorkspaceContextGroupsMainAndLinkedWorktrees(t *testing.T) {
	ctx := context.Background()
	workspaceDirectory := t.TempDir()
	mainPath := initialiseRepository(t, workspaceDirectory, "wade")
	linkedPath := filepath.Join(workspaceDirectory, "wade-feature")
	runRepositoryGit(t, mainPath, "remote", "add", "origin", "git@github.com:example/wade.git")
	runRepositoryGit(t, mainPath, "worktree", "add", "-b", "feature/example", linkedPath)

	model := newTestModel(workspaceDirectory)
	workspaceContext, err := model.GetWorkspaceContext(ctx, "wade-feature")
	if err != nil {
		t.Fatalf("GetWorkspaceContext() error = %v", err)
	}
	if workspaceContext == nil {
		t.Fatal("GetWorkspaceContext() = nil")
	}
	if workspaceContext.Repository.ID != "wade" || workspaceContext.Repository.MainWorkspaceID != "wade" {
		t.Fatalf("Repository = %#v", workspaceContext.Repository)
	}
	if !reflect.DeepEqual(workspaceContext.Repository.WorkspaceIDs, []string{"wade", "wade-feature"}) {
		t.Fatalf("WorkspaceIDs = %#v", workspaceContext.Repository.WorkspaceIDs)
	}
	if workspaceContext.Repository.RemoteRepositoryID == nil || *workspaceContext.Repository.RemoteRepositoryID != "example/wade" {
		t.Fatalf("RemoteRepositoryID = %#v", workspaceContext.Repository.RemoteRepositoryID)
	}
	if workspaceContext.IsMain || !workspaceContext.IsRemovable || workspaceContext.Branch.Ref != "refs/heads/feature/example" {
		t.Fatalf("workspace context = %#v", workspaceContext)
	}

	mainContext, err := model.GetWorkspaceContext(ctx, "wade")
	if err != nil || mainContext == nil || !mainContext.IsMain || mainContext.IsRemovable {
		t.Fatalf("main context = %#v, error = %v", mainContext, err)
	}
}

func TestGetWorkspaceContextReturnsDetachedHeadState(t *testing.T) {
	workspaceDirectory := t.TempDir()
	repositoryPath := initialiseRepository(t, workspaceDirectory, "detached")
	runRepositoryGit(t, repositoryPath, "checkout", "--detach", "HEAD")

	workspaceContext, err := newTestModel(workspaceDirectory).GetWorkspaceContext(context.Background(), "detached")
	if err != nil || workspaceContext == nil {
		t.Fatalf("GetWorkspaceContext() = %#v, error = %v", workspaceContext, err)
	}
	if !workspaceContext.Branch.IsDetached || workspaceContext.Branch.Ref != "" || workspaceContext.Branch.Commit == "" {
		t.Fatalf("Branch = %#v", workspaceContext.Branch)
	}
}

func TestGetWorkspaceContextReturnsNilForNonGitWorkspace(t *testing.T) {
	workspaceDirectory := t.TempDir()
	if err := os.Mkdir(filepath.Join(workspaceDirectory, "notes"), 0o755); err != nil {
		t.Fatal(err)
	}

	workspaceContext, err := newTestModel(workspaceDirectory).GetWorkspaceContext(context.Background(), "notes")
	if err != nil || workspaceContext != nil {
		t.Fatalf("GetWorkspaceContext() = %#v, error = %v", workspaceContext, err)
	}
}

func TestListWorkspaceContextsInspectsWorkspacesWithBoundedConcurrency(t *testing.T) {
	workspaceDirectory := t.TempDir()
	workspaceCount := workspaceInspectionConcurrency + 4
	for index := range workspaceCount {
		workspacePath := filepath.Join(workspaceDirectory, fmt.Sprintf("workspace-%02d", index))
		if err := os.Mkdir(workspacePath, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	releaseInspections := make(chan struct{})
	gitClient := &inspectionConcurrencyGit{
		Client:  git.NewClient(),
		started: make(chan struct{}, workspaceCount),
		release: releaseInspections,
	}
	model := New(
		filesystem.NewWorkspaceDiscovery([]string{workspaceDirectory}),
		gitClient,
		filesystem.NewFileSystem(),
		Configuration{},
	)

	inspectionComplete := make(chan error, 1)
	go func() {
		_, err := model.ListWorkspaceContexts(ctx)
		inspectionComplete <- err
	}()

	for range workspaceInspectionConcurrency {
		select {
		case <-gitClient.started:
		case <-time.After(time.Second):
			t.Fatal("workspace inspections did not run concurrently")
		}
	}
	select {
	case <-gitClient.started:
		t.Fatalf("more than %d workspace inspections ran concurrently", workspaceInspectionConcurrency)
	case <-time.After(100 * time.Millisecond):
	}

	close(releaseInspections)
	select {
	case err := <-inspectionComplete:
		if err != nil {
			t.Fatalf("ListWorkspaceContexts() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("workspace inspections did not complete")
	}
	if maximumConcurrency := gitClient.maximumConcurrency(); maximumConcurrency != workspaceInspectionConcurrency {
		t.Fatalf("maximum inspection concurrency = %d, want %d", maximumConcurrency, workspaceInspectionConcurrency)
	}
}

func TestWorkspaceIDsByRemoteRepositoryCombinesIndependentClones(t *testing.T) {
	workspaceDirectory := t.TempDir()
	firstPath := initialiseRepository(t, workspaceDirectory, "wade-first")
	secondPath := initialiseRepository(t, workspaceDirectory, "wade-second")
	for _, repositoryPath := range []string{firstPath, secondPath} {
		runRepositoryGit(t, repositoryPath, "remote", "add", "origin", "https://github.com/example/wade.git")
	}

	workspaceIDs, err := newTestModel(workspaceDirectory).WorkspaceIDsByRemoteRepository(context.Background(), []string{"example/wade"})
	if err != nil {
		t.Fatalf("WorkspaceIDsByRemoteRepository() error = %v", err)
	}
	if !reflect.DeepEqual(workspaceIDs["example/wade"], []string{"wade-first", "wade-second"}) {
		t.Fatalf("workspace IDs = %#v", workspaceIDs)
	}
}

type inspectionConcurrencyGit struct {
	git.Client

	mu                 sync.Mutex
	activeInspections  int
	maximumInspections int
	started            chan struct{}
	release            <-chan struct{}
}

func (client *inspectionConcurrencyGit) IsGitWorktree(ctx context.Context, _ string) (bool, error) {
	client.mu.Lock()
	client.activeInspections++
	client.maximumInspections = max(client.maximumInspections, client.activeInspections)
	client.mu.Unlock()

	client.started <- struct{}{}
	defer func() {
		client.mu.Lock()
		client.activeInspections--
		client.mu.Unlock()
	}()

	select {
	case <-client.release:
		return false, nil
	case <-ctx.Done():
		return false, ctx.Err()
	}
}

func (client *inspectionConcurrencyGit) maximumConcurrency() int {
	client.mu.Lock()
	defer client.mu.Unlock()
	return client.maximumInspections
}

func newTestModel(workspaceDirectory string) *Model {
	return New(
		filesystem.NewWorkspaceDiscovery([]string{workspaceDirectory}),
		git.NewClient(),
		filesystem.NewFileSystem(),
		Configuration{},
	)
}

func initialiseRepository(t *testing.T, workspaceDirectory string, name string) string {
	t.Helper()
	repositoryPath := filepath.Join(workspaceDirectory, name)
	if err := os.Mkdir(repositoryPath, 0o755); err != nil {
		t.Fatalf("Mkdir(%q): %v", repositoryPath, err)
	}
	runRepositoryGit(t, repositoryPath, "init", "-b", "main")
	runRepositoryGit(t, repositoryPath, "config", "user.email", "test@example.com")
	runRepositoryGit(t, repositoryPath, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(repositoryPath, "README.md"), []byte("# Test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runRepositoryGit(t, repositoryPath, "add", "README.md")
	runRepositoryGit(t, repositoryPath, "commit", "-m", "initial commit")
	return repositoryPath
}

func runRepositoryGit(t *testing.T, repositoryPath string, args ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", repositoryPath}, args...)...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git -C %q %s: %v, output = %s", repositoryPath, strings.Join(args, " "), err, string(output))
	}
}
