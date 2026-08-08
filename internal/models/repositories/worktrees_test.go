package repositories

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"wade/internal/infrastructure/filesystem"
	"wade/internal/infrastructure/git"
)

func TestCreateWorktreeReturnsRepositoryScopedWorktree(t *testing.T) {
	projectPath := initGitRepository(t)
	model := newWorktreeTestModel(projectPath)

	created, err := model.CreateWorktree(context.Background(), "project", CreateWorktreeRequest{BranchRef: "refs/heads/feature/example"})
	if err != nil {
		t.Fatalf("CreateWorktree() error = %v", err)
	}
	if created.ID != "project-feature-example" || created.WorkspaceID != created.ID || created.RepositoryID != "project" {
		t.Fatalf("created worktree = %#v", created)
	}
	if created.Branch == nil || created.Branch.Ref != "refs/heads/feature/example" || created.IsMain || !created.IsRemovable {
		t.Fatalf("created worktree = %#v", created)
	}
}

func TestListBranchesReturnsEmptyForRepositoryWithoutRemote(t *testing.T) {
	model := newWorktreeTestModel(initGitRepository(t))
	branches, err := model.ListBranches(context.Background(), "project", BranchKindRemote)
	if err != nil || len(branches) != 0 {
		t.Fatalf("ListBranches() = %#v, error = %v", branches, err)
	}
}

func TestRemoteBranchCanCreateWorktree(t *testing.T) {
	ctx := context.Background()
	projectPath := initGitRepository(t)
	remotePath := filepath.Join(t.TempDir(), "origin.git")
	runGit(t, filepath.Dir(remotePath), "init", "--bare", remotePath)
	runGit(t, projectPath, "remote", "add", "origin", remotePath)
	runGit(t, projectPath, "checkout", "-b", "feature/remote")
	if err := os.WriteFile(filepath.Join(projectPath, "feature.txt"), []byte("feature\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, projectPath, "add", "feature.txt")
	runGit(t, projectPath, "commit", "-m", "feature")
	runGit(t, projectPath, "push", "-u", "origin", "feature/remote")
	runGit(t, projectPath, "checkout", "main")
	runGit(t, projectPath, "branch", "-D", "feature/remote")

	model := newWorktreeTestModel(projectPath)
	branches, err := model.ListBranches(ctx, "project", BranchKindRemote)
	if err != nil {
		t.Fatalf("ListBranches() error = %v", err)
	}
	var remoteBranch *Branch
	for index := range branches {
		if branches[index].Name == "feature/remote" {
			remoteBranch = &branches[index]
		}
	}
	if remoteBranch == nil || remoteBranch.Ref != "refs/remotes/origin/feature/remote" || remoteBranch.HasLocalBranch {
		t.Fatalf("remote branch = %#v", remoteBranch)
	}

	created, err := model.CreateWorktree(ctx, "project", CreateWorktreeRequest{BranchRef: remoteBranch.Ref})
	if err != nil || created.Branch == nil || created.Branch.Name != "feature/remote" {
		t.Fatalf("CreateWorktree() = %#v, error = %v", created, err)
	}
}

func TestCreateWorktreeRejectsInvalidBranchReferences(t *testing.T) {
	projectPath := initGitRepository(t)
	remotePath := filepath.Join(t.TempDir(), "origin.git")
	runGit(t, filepath.Dir(remotePath), "init", "--bare", remotePath)
	runGit(t, projectPath, "remote", "add", "origin", remotePath)
	model := newWorktreeTestModel(projectPath)

	for name, branchRef := range map[string]string{
		"invalid branch name":       "refs/heads/feature..invalid",
		"malformed remote ref":      "refs/remotes/origin",
		"missing remote":            "refs/remotes/upstream/feature/example",
		"missing remote branch":     "refs/remotes/origin/feature/missing",
		"unsupported ref namespace": "refs/tags/release",
	} {
		t.Run(name, func(t *testing.T) {
			_, err := model.CreateWorktree(context.Background(), "project", CreateWorktreeRequest{BranchRef: branchRef})
			var invalidReference InvalidBranchReferenceError
			if !errors.As(err, &invalidReference) || invalidReference.BranchRef != branchRef {
				t.Fatalf("CreateWorktree() error = %v", err)
			}
		})
	}
}

func TestRemoveWorktreeDeletesLocalBranch(t *testing.T) {
	ctx := context.Background()
	projectPath := initGitRepository(t)
	model := newWorktreeTestModel(projectPath)
	created, err := model.CreateWorktree(ctx, "project", CreateWorktreeRequest{BranchRef: "remove-me"})
	if err != nil {
		t.Fatal(err)
	}

	removed, err := model.RemoveWorktree(ctx, "project", created.ID)
	if err != nil || removed.ID != created.ID {
		t.Fatalf("RemoveWorktree() = %#v, error = %v", removed, err)
	}
	if output := gitOutputForTest(t, projectPath, "branch", "--list", "remove-me"); output != "" {
		t.Fatalf("branch still exists: %q", output)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(projectPath), created.ID)); !os.IsNotExist(err) {
		t.Fatalf("removed path Stat() error = %v", err)
	}
}

func TestRemoveWorktreeRejectsMainWorktree(t *testing.T) {
	model := newWorktreeTestModel(initGitRepository(t))
	_, err := model.RemoveWorktree(context.Background(), "project", "project")
	var notRemovable WorktreeNotRemovableError
	if !errors.As(err, &notRemovable) {
		t.Fatalf("RemoveWorktree() error = %v", err)
	}
}

func newWorktreeTestModel(projectPath string) *Model {
	return New(
		filesystem.NewWorkspaceDiscovery([]string{filepath.Dir(projectPath)}),
		git.NewClient(),
		filesystem.NewFileSystem(),
		Configuration{},
	)
}

func initGitRepository(t *testing.T) string {
	t.Helper()
	projectPath := filepath.Join(t.TempDir(), "project")
	if err := os.Mkdir(projectPath, 0o755); err != nil {
		t.Fatal(err)
	}
	runGit(t, projectPath, "init", "-b", "main")
	runGit(t, projectPath, "config", "user.email", "test@example.com")
	runGit(t, projectPath, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(projectPath, "README.md"), []byte("# Test\n"), 0o644); err != nil {
		t.Fatal(err)
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
		t.Fatalf("git -C %q %s: %v, output = %s", projectPath, strings.Join(args, " "), err, string(output))
	}
	return strings.TrimSpace(string(output))
}
