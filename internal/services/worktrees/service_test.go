package worktrees

// TODO: Review properly

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"wade/internal/repositories"
	"wade/internal/services/config"
)

func TestRemoveDeletesLocalBranch(t *testing.T) {
	ctx := context.Background()
	projectPath := initGitRepository(t)
	service := NewService(config.Config{}, repositories.NewGitRepository(), repositories.NewFileRepository())
	branchName := "remove-me"

	created, err := service.Create(ctx, projectPath, branchName)
	if err != nil {
		t.Fatalf("Create() error = %v, want nil", err)
	}

	if err := service.Remove(ctx, projectPath, created); err != nil {
		t.Fatalf("Remove() error = %v, want nil", err)
	}

	if output := gitOutputForTest(t, projectPath, "branch", "--list", branchName); output != "" {
		t.Fatalf("local branch exists after Remove() = %q, want empty", output)
	}

	if _, err := os.Stat(created.Path); !os.IsNotExist(err) {
		t.Fatalf("removed worktree path Stat() error = %v, want not exist", err)
	}
}

func TestRemoveDetachedWorktreeDoesNotDeleteBranch(t *testing.T) {
	ctx := context.Background()
	projectPath := initGitRepository(t)
	service := NewService(config.Config{}, repositories.NewGitRepository(), repositories.NewFileRepository())
	detachedPath := filepath.Join(filepath.Dir(projectPath), "project-detached")

	runGit(t, projectPath, "worktree", "add", "--detach", detachedPath, "HEAD")
	target, err := service.Find(ctx, projectPath, detachedPath)
	if err != nil {
		t.Fatalf("Find() error = %v, want nil", err)
	}

	if target.Branch != "" {
		t.Fatalf("detached worktree Branch = %q, want empty", target.Branch)
	}

	if err := service.Remove(ctx, projectPath, target); err != nil {
		t.Fatalf("Remove() error = %v, want nil", err)
	}

	if output := gitOutputForTest(t, projectPath, "branch", "--list", "main"); !strings.Contains(output, "main") {
		t.Fatalf("main branch after Remove() = %q, want main", output)
	}

	if _, err := os.Stat(detachedPath); !os.IsNotExist(err) {
		t.Fatalf("removed detached worktree path Stat() error = %v, want not exist", err)
	}
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
