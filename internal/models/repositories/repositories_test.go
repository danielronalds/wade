package repositories

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

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
