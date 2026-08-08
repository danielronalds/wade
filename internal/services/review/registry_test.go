package review

import (
	"context"
	"errors"
	"testing"

	"wade/internal/infrastructure/filesystem"
	"wade/internal/infrastructure/git"
)

type reviewWorkspaceRepositoryStub struct {
	path string
}

func (s reviewWorkspaceRepositoryStub) Resolve(string) (string, bool, error) {
	return s.path, true, nil
}

func TestSnapshotLifecycle(t *testing.T) {
	repoRoot := t.TempDir()
	runGitCommand(t, repoRoot, "init", "-b", "main")
	runGitCommand(t, repoRoot, "config", "user.email", "wade@example.com")
	runGitCommand(t, repoRoot, "config", "user.name", "WADE")
	writeReviewTestFile(t, repoRoot, "tracked.txt", "old\n")
	runGitCommand(t, repoRoot, "add", "tracked.txt")
	runGitCommand(t, repoRoot, "commit", "-m", "initial")
	writeReviewTestFile(t, repoRoot, "tracked.txt", "new\n")

	service := NewService(
		reviewWorkspaceRepositoryStub{path: repoRoot},
		git.NewClient(),
		nil,
		filesystem.NewFileSystem(),
	)

	snapshot, err := service.CreateSnapshot(context.Background(), "wade")
	if err != nil {
		t.Fatalf("CreateSnapshot() error = %v, want nil", err)
	}
	if snapshot.ID == "" || snapshot.WorkspaceID != "wade" {
		t.Fatalf("CreateSnapshot() identity = %q/%q, want generated/wade", snapshot.ID, snapshot.WorkspaceID)
	}
	if snapshot.Branch == nil || snapshot.Branch.Ref != "refs/heads/main" {
		t.Fatalf("CreateSnapshot() Branch = %#v, want main", snapshot.Branch)
	}

	file := findReviewFile(t, snapshot.Files, "tracked.txt")
	runGitCommand(t, repoRoot, "add", "tracked.txt")
	runGitCommand(t, repoRoot, "commit", "-m", "advance head after snapshot")

	contents, err := service.LoadSnapshotFileContents(
		context.Background(),
		snapshot.ID,
		file.ID,
		ScopeWorkingTree,
	)
	if err != nil {
		t.Fatalf("LoadSnapshotFileContents() error = %v, want nil", err)
	}
	if contents.OriginalContent != "old\n" || contents.ModifiedContent != "new\n" {
		t.Fatalf("contents = %#v, want old/new", contents)
	}

	_, err = service.LoadSnapshotFileContents(context.Background(), snapshot.ID, "unknown", ScopeCurrent)
	var fileNotFoundError SnapshotFileNotFoundError
	if !errors.As(err, &fileNotFoundError) {
		t.Fatalf("LoadSnapshotFileContents() error = %v, want SnapshotFileNotFoundError", err)
	}

	if err := service.DeleteSnapshot(snapshot.ID); err != nil {
		t.Fatalf("DeleteSnapshot() error = %v, want nil", err)
	}
	if _, err := service.GetSnapshot(snapshot.ID); err == nil {
		t.Fatal("GetSnapshot() error = nil after deletion, want not found")
	}
}

func TestSnapshotRegistryIsEphemeralAcrossServiceRestart(t *testing.T) {
	repoRoot := t.TempDir()
	runGitCommand(t, repoRoot, "init", "-b", "main")
	runGitCommand(t, repoRoot, "config", "user.email", "wade@example.com")
	runGitCommand(t, repoRoot, "config", "user.name", "WADE")
	writeReviewTestFile(t, repoRoot, "tracked.txt", "content\n")
	runGitCommand(t, repoRoot, "add", "tracked.txt")
	runGitCommand(t, repoRoot, "commit", "-m", "initial")

	workspaceRepository := reviewWorkspaceRepositoryStub{path: repoRoot}
	service := NewService(workspaceRepository, git.NewClient(), nil, filesystem.NewFileSystem())
	snapshot, err := service.CreateSnapshot(context.Background(), "wade")
	if err != nil {
		t.Fatalf("CreateSnapshot() error = %v, want nil", err)
	}

	restartedService := NewService(workspaceRepository, git.NewClient(), nil, filesystem.NewFileSystem())
	_, err = restartedService.GetSnapshot(snapshot.ID)
	var notFoundError SnapshotNotFoundError
	if !errors.As(err, &notFoundError) {
		t.Fatalf("GetSnapshot() error = %v, want SnapshotNotFoundError", err)
	}
}
