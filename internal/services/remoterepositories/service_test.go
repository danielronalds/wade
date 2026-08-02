package remoterepositories

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"wade/internal/repositories"
	"wade/internal/services/gitrepositories"
	"wade/internal/services/workspaces"
)

type gitHubRepositoryStub struct {
	listOutput       string
	clonedRepository string
	cloneTarget      string
}

func (s *gitHubRepositoryStub) ListRepositories(context.Context) (string, error) {
	return s.listOutput, nil
}

func (s *gitHubRepositoryStub) CloneRepository(_ context.Context, repositoryID string, targetPath string) error {
	s.clonedRepository = repositoryID
	s.cloneTarget = targetPath
	return os.Mkdir(targetPath, 0o755)
}

type workspaceServiceStub struct {
	workspace workspaces.Workspace
}

func (s workspaceServiceStub) Get(context.Context, string) (workspaces.Workspace, error) {
	return s.workspace, nil
}

func TestListMatchesLocalRepositoriesByCanonicalRemote(t *testing.T) {
	workspaceDirectory := t.TempDir()
	repositoryPath := initialiseRemoteTestRepository(t, workspaceDirectory, "wade")
	runRemoteTestGit(t, repositoryPath, "remote", "add", "origin", "https://github.com/Example/WADE.git")

	workspaceRepository := repositories.NewWorkspaceStore([]string{workspaceDirectory})
	localRepositories := gitrepositories.NewService(workspaceRepository, repositories.NewGitRepository())
	github := &gitHubRepositoryStub{listOutput: `[{"name":"wade","nameWithOwner":"example/wade","url":"https://github.com/example/wade","sshUrl":"git@github.com:example/wade.git"}]`}
	service := NewService(github, repositories.NewFileRepository(), localRepositories, workspaceRepository, nil, nil)

	got, err := service.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v, want nil", err)
	}
	if len(got) != 1 {
		t.Fatalf("List() length = %d, want 1", len(got))
	}
	if len(got[0].LocalWorkspaceIDs) != 1 || got[0].LocalWorkspaceIDs[0] != "wade" {
		t.Fatalf("List() LocalWorkspaceIDs = %#v, want [wade]", got[0].LocalWorkspaceIDs)
	}
}

func TestCloneUsesExactConfiguredWorkspaceDirectory(t *testing.T) {
	workspaceDirectory := t.TempDir()
	workspaceRepository := repositories.NewWorkspaceStore([]string{workspaceDirectory})
	github := &gitHubRepositoryStub{}
	workspaceService := workspaceServiceStub{workspace: workspaces.Workspace{ID: "wade", Name: "wade"}}
	service := NewService(
		github,
		repositories.NewFileRepository(),
		gitrepositories.NewService(workspaceRepository, repositories.NewGitRepository()),
		workspaceRepository,
		workspaceService,
		[]WorkspaceDirectory{{Setting: "~/Code", Path: workspaceDirectory}},
	)

	got, err := service.Clone(context.Background(), CloneRequest{
		RemoteRepositoryID: "example/wade",
		WorkspaceDirectory: "~/Code",
	})
	if err != nil {
		t.Fatalf("Clone() error = %v, want nil", err)
	}
	if got.ID != "wade" {
		t.Fatalf("Clone() workspace ID = %q, want wade", got.ID)
	}
	if github.clonedRepository != "example/wade" {
		t.Fatalf("CloneRepository() repository = %q, want example/wade", github.clonedRepository)
	}
	wantTarget := filepath.Join(workspaceDirectory, "wade")
	if github.cloneTarget != wantTarget {
		t.Fatalf("CloneRepository() target = %q, want %q", github.cloneTarget, wantTarget)
	}
}

func TestCloneRejectsResolvedPathInsteadOfConfiguredString(t *testing.T) {
	workspaceDirectory := t.TempDir()
	workspaceRepository := repositories.NewWorkspaceStore([]string{workspaceDirectory})
	service := NewService(
		&gitHubRepositoryStub{},
		repositories.NewFileRepository(),
		gitrepositories.NewService(workspaceRepository, repositories.NewGitRepository()),
		workspaceRepository,
		workspaceServiceStub{},
		[]WorkspaceDirectory{{Setting: "~/Code", Path: workspaceDirectory}},
	)

	_, err := service.Clone(context.Background(), CloneRequest{
		RemoteRepositoryID: "example/wade",
		WorkspaceDirectory: workspaceDirectory,
	})

	var notConfiguredError WorkspaceDirectoryNotConfiguredError
	if !errors.As(err, &notConfiguredError) {
		t.Fatalf("Clone() error = %v, want WorkspaceDirectoryNotConfiguredError", err)
	}
}

func initialiseRemoteTestRepository(t *testing.T, workspaceDirectory string, name string) string {
	t.Helper()

	repositoryPath := filepath.Join(workspaceDirectory, name)
	if err := os.Mkdir(repositoryPath, 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v, want nil", err)
	}
	runRemoteTestGit(t, repositoryPath, "init", "-b", "main")
	runRemoteTestGit(t, repositoryPath, "config", "user.email", "test@example.com")
	runRemoteTestGit(t, repositoryPath, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(repositoryPath, "README.md"), []byte("# Test\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v, want nil", err)
	}
	runRemoteTestGit(t, repositoryPath, "add", "README.md")
	runRemoteTestGit(t, repositoryPath, "commit", "-m", "initial commit")
	return repositoryPath
}

func runRemoteTestGit(t *testing.T, repositoryPath string, args ...string) {
	t.Helper()

	command := exec.Command("git", append([]string{"-C", repositoryPath}, args...)...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git -C %q %s error = %v, output = %s", repositoryPath, strings.Join(args, " "), err, string(output))
	}
}
