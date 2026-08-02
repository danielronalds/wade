package gitrepositories

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"wade/internal/repositories"
)

func TestResolveWorkspaceGroupsMainAndLinkedWorktrees(t *testing.T) {
	ctx := context.Background()
	workspaceDirectory := t.TempDir()
	mainPath := initialiseRepository(t, workspaceDirectory, "wade")
	linkedPath := filepath.Join(workspaceDirectory, "wade-feature")
	runRepositoryGit(t, mainPath, "remote", "add", "origin", "git@github.com:example/wade.git")
	runRepositoryGit(t, mainPath, "worktree", "add", "-b", "feature/example", linkedPath)

	workspaceRepository := repositories.NewWorkspaceStore([]string{workspaceDirectory})
	service := NewService(workspaceRepository, repositories.NewGitRepository())

	workspaceContext, isGit, err := service.ResolveWorkspace(ctx, "wade-feature")
	if err != nil {
		t.Fatalf("ResolveWorkspace() error = %v, want nil", err)
	}
	if !isGit {
		t.Fatal("ResolveWorkspace() isGit = false, want true")
	}

	repository := workspaceContext.RepositoryContext.Repository
	if repository.ID != "wade" {
		t.Fatalf("Repository.ID = %q, want wade", repository.ID)
	}
	if repository.MainWorkspaceID != "wade" {
		t.Fatalf("Repository.MainWorkspaceID = %q, want wade", repository.MainWorkspaceID)
	}
	wantWorkspaceIDs := []string{"wade", "wade-feature"}
	if !reflect.DeepEqual(repository.WorkspaceIDs, wantWorkspaceIDs) {
		t.Fatalf("Repository.WorkspaceIDs = %#v, want %#v", repository.WorkspaceIDs, wantWorkspaceIDs)
	}
	if repository.RemoteRepositoryID == nil || *repository.RemoteRepositoryID != "example/wade" {
		t.Fatalf("Repository.RemoteRepositoryID = %#v, want example/wade", repository.RemoteRepositoryID)
	}
	if workspaceContext.IsMain || !workspaceContext.IsRemovable {
		t.Fatalf("linked worktree main/removable = %v/%v, want false/true", workspaceContext.IsMain, workspaceContext.IsRemovable)
	}
	if workspaceContext.Branch.Ref != "refs/heads/feature/example" || workspaceContext.Branch.Commit == "" {
		t.Fatalf("linked worktree Branch = %#v, want feature branch with commit", workspaceContext.Branch)
	}

	mainContext, isGit, err := service.ResolveWorkspace(ctx, "wade")
	if err != nil {
		t.Fatalf("ResolveWorkspace(main) error = %v, want nil", err)
	}
	if !isGit || !mainContext.IsMain || mainContext.IsRemovable {
		t.Fatalf("main worktree git/main/removable = %v/%v/%v, want true/true/false", isGit, mainContext.IsMain, mainContext.IsRemovable)
	}

	resolvedRepository, err := service.Resolve(ctx, "wade")
	if err != nil {
		t.Fatalf("Resolve() error = %v, want nil", err)
	}
	if resolvedRepository.CommonDirectory() != workspaceContext.RepositoryContext.CommonDirectory() {
		t.Fatalf("Resolve() common directory = %q, want %q", resolvedRepository.CommonDirectory(), workspaceContext.RepositoryContext.CommonDirectory())
	}
}

func TestResolveWorkspaceReturnsDetachedHeadState(t *testing.T) {
	workspaceDirectory := t.TempDir()
	repositoryPath := initialiseRepository(t, workspaceDirectory, "detached")
	runRepositoryGit(t, repositoryPath, "checkout", "--detach", "HEAD")

	workspaceRepository := repositories.NewWorkspaceStore([]string{workspaceDirectory})
	service := NewService(workspaceRepository, repositories.NewGitRepository())

	workspaceContext, isGit, err := service.ResolveWorkspace(context.Background(), "detached")
	if err != nil {
		t.Fatalf("ResolveWorkspace() error = %v, want nil", err)
	}
	if !isGit {
		t.Fatal("ResolveWorkspace() isGit = false, want true")
	}
	if !workspaceContext.Branch.IsDetached || workspaceContext.Branch.Ref != "" || workspaceContext.Branch.Commit == "" {
		t.Fatalf("Branch = %#v, want detached branch with commit", workspaceContext.Branch)
	}
}

func TestResolveWorkspaceSupportsRepositoryWithoutCommits(t *testing.T) {
	workspaceDirectory := t.TempDir()
	repositoryPath := filepath.Join(workspaceDirectory, "empty")
	if err := os.Mkdir(repositoryPath, 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v, want nil", err)
	}
	runRepositoryGit(t, repositoryPath, "init", "-b", "main")

	workspaceRepository := repositories.NewWorkspaceStore([]string{workspaceDirectory})
	service := NewService(workspaceRepository, repositories.NewGitRepository())

	workspaceContext, isGit, err := service.ResolveWorkspace(context.Background(), "empty")
	if err != nil {
		t.Fatalf("ResolveWorkspace() error = %v, want nil", err)
	}
	if !isGit {
		t.Fatal("ResolveWorkspace() isGit = false, want true")
	}
	if workspaceContext.Branch.Ref != "refs/heads/main" || workspaceContext.Branch.Commit != "" {
		t.Fatalf("Branch = %#v, want unborn main branch", workspaceContext.Branch)
	}
}

func TestResolveWorkspaceDetectsNonGitWorkspace(t *testing.T) {
	workspaceDirectory := t.TempDir()
	notesPath := filepath.Join(workspaceDirectory, "notes")
	if err := os.Mkdir(notesPath, 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v, want nil", err)
	}

	workspaceRepository := repositories.NewWorkspaceStore([]string{workspaceDirectory})
	service := NewService(workspaceRepository, repositories.NewGitRepository())

	_, isGit, err := service.ResolveWorkspace(context.Background(), "notes")
	if err != nil {
		t.Fatalf("ResolveWorkspace() error = %v, want nil", err)
	}
	if isGit {
		t.Fatal("ResolveWorkspace() isGit = true, want false")
	}
}

func TestListKeepsIndependentClonesAsSeparateLocalRepositories(t *testing.T) {
	workspaceDirectory := t.TempDir()
	firstPath := initialiseRepository(t, workspaceDirectory, "wade-first")
	secondPath := initialiseRepository(t, workspaceDirectory, "wade-second")
	for _, repositoryPath := range []string{firstPath, secondPath} {
		runRepositoryGit(t, repositoryPath, "remote", "add", "origin", "https://github.com/example/wade.git")
	}

	workspaceRepository := repositories.NewWorkspaceStore([]string{workspaceDirectory})
	service := NewService(workspaceRepository, repositories.NewGitRepository())

	got, err := service.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v, want nil", err)
	}
	if len(got) != 2 {
		t.Fatalf("List() length = %d, want 2", len(got))
	}
	if got[0].Repository.ID != "wade-first" || got[1].Repository.ID != "wade-second" {
		t.Fatalf("List() repository IDs = %q/%q, want wade-first/wade-second", got[0].Repository.ID, got[1].Repository.ID)
	}
	for _, repository := range got {
		if repository.Repository.RemoteRepositoryID == nil || *repository.Repository.RemoteRepositoryID != "example/wade" {
			t.Fatalf("RemoteRepositoryID = %#v, want example/wade", repository.Repository.RemoteRepositoryID)
		}
	}
	if got[0].CommonDirectory() == got[1].CommonDirectory() {
		t.Fatalf("independent clone common directories are both %q, want different", got[0].CommonDirectory())
	}
}

func initialiseRepository(t *testing.T, workspaceDirectory string, name string) string {
	t.Helper()

	repositoryPath := filepath.Join(workspaceDirectory, name)
	if err := os.Mkdir(repositoryPath, 0o755); err != nil {
		t.Fatalf("Mkdir(%q) error = %v, want nil", repositoryPath, err)
	}

	runRepositoryGit(t, repositoryPath, "init", "-b", "main")
	runRepositoryGit(t, repositoryPath, "config", "user.email", "test@example.com")
	runRepositoryGit(t, repositoryPath, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(repositoryPath, "README.md"), []byte("# Test\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v, want nil", err)
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
		t.Fatalf("git -C %q %s error = %v, output = %s", repositoryPath, strings.Join(args, " "), err, string(output))
	}
}
