package worktrees

// TODO: Review properly

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"wade/internal/repositories"
	"wade/internal/services/config"
	"wade/internal/services/gitrepositories"
)

type terminalServiceStub struct {
	closedDirectory string
}

func (s *terminalServiceStub) CloseTerminalsForDirectory(directory string) int {
	s.closedDirectory = directory
	return 1
}

func TestCreateReturnsRepositoryScopedWorktree(t *testing.T) {
	ctx := context.Background()
	projectPath := initGitRepository(t)
	service, repository := newTestService(t, ctx, projectPath, nil)

	created, err := service.Create(ctx, repository, "refs/heads/feature/example")
	if err != nil {
		t.Fatalf("Create() error = %v, want nil", err)
	}

	if created.ID != "project-feature-example" || created.WorkspaceID != created.ID {
		t.Fatalf("Create() identity = %q/%q, want project-feature-example", created.ID, created.WorkspaceID)
	}
	if created.RepositoryID != "project" {
		t.Fatalf("Create() RepositoryID = %q, want project", created.RepositoryID)
	}
	if created.Branch == nil || created.Branch.Ref != "refs/heads/feature/example" {
		t.Fatalf("Create() Branch = %#v, want refs/heads/feature/example", created.Branch)
	}
	if created.IsMain || !created.IsRemovable {
		t.Fatalf("Create() main/removable = %v/%v, want false/true", created.IsMain, created.IsRemovable)
	}
}

func TestRemoteBranchesReturnsEmptyForRepositoryWithoutRemote(t *testing.T) {
	ctx := context.Background()
	projectPath := initGitRepository(t)
	service, repository := newTestService(t, ctx, projectPath, nil)

	branches, err := service.Branches(ctx, repository, BranchKindRemote)
	if err != nil {
		t.Fatalf("Branches() error = %v, want nil", err)
	}
	if len(branches) != 0 {
		t.Fatalf("Branches() = %#v, want empty", branches)
	}
}

func TestRepositoryScopedRemoteBranchCanCreateWorktree(t *testing.T) {
	ctx := context.Background()
	projectPath := initGitRepository(t)
	remotePath := filepath.Join(t.TempDir(), "origin.git")
	runGit(t, filepath.Dir(remotePath), "init", "--bare", remotePath)
	runGit(t, projectPath, "remote", "add", "origin", remotePath)
	runGit(t, projectPath, "checkout", "-b", "feature/remote")
	if err := os.WriteFile(filepath.Join(projectPath, "feature.txt"), []byte("feature\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v, want nil", err)
	}
	runGit(t, projectPath, "add", "feature.txt")
	runGit(t, projectPath, "commit", "-m", "feature")
	runGit(t, projectPath, "push", "-u", "origin", "feature/remote")
	runGit(t, projectPath, "checkout", "main")
	runGit(t, projectPath, "branch", "-D", "feature/remote")

	service, repository := newTestService(t, ctx, projectPath, nil)
	branches, err := service.Branches(ctx, repository, BranchKindRemote)
	if err != nil {
		t.Fatalf("Branches() error = %v, want nil", err)
	}

	var remoteBranch *Branch
	for index := range branches {
		if branches[index].Name == "feature/remote" {
			remoteBranch = &branches[index]
			break
		}
	}
	if remoteBranch == nil {
		t.Fatalf("Branches() = %#v, want feature/remote", branches)
	}
	if remoteBranch.Ref != "refs/remotes/origin/feature/remote" || remoteBranch.HasLocalBranch {
		t.Fatalf("remote branch = %#v, want remote-only full ref", remoteBranch)
	}

	created, err := service.Create(ctx, repository, remoteBranch.Ref)
	if err != nil {
		t.Fatalf("Create() error = %v, want nil", err)
	}
	if created.Branch == nil || created.Branch.Name != "feature/remote" {
		t.Fatalf("Create() Branch = %#v, want feature/remote", created.Branch)
	}
}

func TestCreateRejectsInvalidBranchReferences(t *testing.T) {
	ctx := context.Background()
	projectPath := initGitRepository(t)
	remotePath := filepath.Join(t.TempDir(), "origin.git")
	runGit(t, filepath.Dir(remotePath), "init", "--bare", remotePath)
	runGit(t, projectPath, "remote", "add", "origin", remotePath)

	service, repository := newTestService(t, ctx, projectPath, nil)
	tests := map[string]string{
		"invalid branch name":       "refs/heads/feature..invalid",
		"malformed remote ref":      "refs/remotes/origin",
		"missing remote":            "refs/remotes/upstream/feature/example",
		"missing remote branch":     "refs/remotes/origin/feature/missing",
		"unsupported ref namespace": "refs/tags/release",
	}

	for name, branchRef := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := service.Create(ctx, repository, branchRef)

			var invalidBranchReference InvalidBranchReferenceError
			if !errors.As(err, &invalidBranchReference) {
				t.Fatalf("Create() error = %v, want InvalidBranchReferenceError", err)
			}
			if invalidBranchReference.BranchRef != branchRef {
				t.Fatalf("InvalidBranchReferenceError.BranchRef = %q, want %q", invalidBranchReference.BranchRef, branchRef)
			}
		})
	}
}

func TestRemoveClosesTerminalsAndDeletesLocalBranch(t *testing.T) {
	ctx := context.Background()
	projectPath := initGitRepository(t)
	terminals := &terminalServiceStub{}
	service, repository := newTestService(t, ctx, projectPath, terminals)
	branchName := "remove-me"

	created, err := service.Create(ctx, repository, branchName)
	if err != nil {
		t.Fatalf("Create() error = %v, want nil", err)
	}

	removed, err := service.Remove(ctx, repository, created.ID)
	if err != nil {
		t.Fatalf("Remove() error = %v, want nil", err)
	}
	if removed.ID != created.ID {
		t.Fatalf("Remove() ID = %q, want %q", removed.ID, created.ID)
	}
	if terminals.closedDirectory != created.Path() {
		t.Fatalf("closed terminal directory = %q, want %q", terminals.closedDirectory, created.Path())
	}
	if output := gitOutputForTest(t, projectPath, "branch", "--list", branchName); output != "" {
		t.Fatalf("local branch exists after Remove() = %q, want empty", output)
	}
	if _, err := os.Stat(created.Path()); !os.IsNotExist(err) {
		t.Fatalf("removed worktree path Stat() error = %v, want not exist", err)
	}
}

func TestRemoveDetachedWorktreeDoesNotDeleteBranch(t *testing.T) {
	ctx := context.Background()
	projectPath := initGitRepository(t)
	detachedPath := filepath.Join(filepath.Dir(projectPath), "project-detached")
	runGit(t, projectPath, "worktree", "add", "--detach", detachedPath, "HEAD")

	service, repository := newTestService(t, ctx, projectPath, nil)
	target, err := service.Get(ctx, repository, filepath.Base(detachedPath))
	if err != nil {
		t.Fatalf("Get() error = %v, want nil", err)
	}
	if target.Branch != nil {
		t.Fatalf("detached worktree Branch = %#v, want nil", target.Branch)
	}

	if _, err := service.Remove(ctx, repository, target.ID); err != nil {
		t.Fatalf("Remove() error = %v, want nil", err)
	}
	if output := gitOutputForTest(t, projectPath, "branch", "--list", "main"); !strings.Contains(output, "main") {
		t.Fatalf("main branch after Remove() = %q, want main", output)
	}
	if _, err := os.Stat(detachedPath); !os.IsNotExist(err) {
		t.Fatalf("removed detached worktree path Stat() error = %v, want not exist", err)
	}
}

func TestRemoveRejectsMainWorktree(t *testing.T) {
	ctx := context.Background()
	projectPath := initGitRepository(t)
	service, repository := newTestService(t, ctx, projectPath, nil)

	_, err := service.Remove(ctx, repository, filepath.Base(projectPath))

	var notRemovableError WorktreeNotRemovableError
	if !errors.As(err, &notRemovableError) {
		t.Fatalf("Remove() error = %v, want WorktreeNotRemovableError", err)
	}
}

func newTestService(
	t *testing.T,
	ctx context.Context,
	projectPath string,
	terminals terminalService,
) (Service, gitrepositories.Context) {
	t.Helper()

	workspaceRepository := repositories.NewWorkspaceStore([]string{filepath.Dir(projectPath)})
	gitRepository := repositories.NewGitRepository()
	localRepositories := gitrepositories.NewService(workspaceRepository, gitRepository)
	workspaceContext, isGit, err := localRepositories.ResolveWorkspace(ctx, filepath.Base(projectPath))
	if err != nil {
		t.Fatalf("ResolveWorkspace() error = %v, want nil", err)
	}
	if !isGit {
		t.Fatal("ResolveWorkspace() isGit = false, want true")
	}

	service := NewService(
		config.Config{},
		gitRepository,
		repositories.NewFileRepository(),
		workspaceRepository,
		terminals,
	)
	return service, workspaceContext.RepositoryContext
}

func initGitRepository(t *testing.T) string {
	t.Helper()

	projectPath := filepath.Join(t.TempDir(), "project")
	if err := os.Mkdir(projectPath, 0o755); err != nil {
		t.Fatalf("Mkdir(%q) error = %v, want nil", projectPath, err)
	}

	runGit(t, projectPath, "init", "-b", "main")
	runGit(t, projectPath, "config", "user.email", "test@example.com")
	runGit(t, projectPath, "config", "user.name", "Test User")

	readmePath := filepath.Join(projectPath, "README.md")
	if err := os.WriteFile(readmePath, []byte("# Test\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v, want nil", readmePath, err)
	}

	runGit(t, projectPath, "add", "README.md")
	runGit(t, projectPath, "commit", "-m", "initial commit")

	return projectPath
}

func runGit(t *testing.T, projectPath string, args ...string) {
	t.Helper()

	_ = gitOutputForTest(t, projectPath, args...)
}

func gitOutputForTest(t *testing.T, projectPath string, args ...string) string {
	t.Helper()

	command := exec.Command("git", append([]string{"-C", projectPath}, args...)...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git -C %q %s error = %v, output = %s", projectPath, strings.Join(args, " "), err, string(output))
	}

	return strings.TrimSpace(string(output))
}
