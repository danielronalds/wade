package filesystem

// TODO: Review properly

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestWorkspaceDiscoveryIDsReturnsUniqueSortedWorkspaceIDs(t *testing.T) {
	firstDirectory := t.TempDir()
	secondDirectory := t.TempDir()

	mustMkdir(t, filepath.Join(firstDirectory, "bravo"))
	mustMkdir(t, filepath.Join(firstDirectory, "alpha"))
	mustMkdir(t, filepath.Join(firstDirectory, "shared"))
	mustMkdir(t, filepath.Join(secondDirectory, "charlie"))
	mustMkdir(t, filepath.Join(secondDirectory, "shared"))
	mustWriteFile(t, filepath.Join(firstDirectory, "not-a-workspace"))

	store := NewWorkspaceDiscovery([]string{firstDirectory, secondDirectory})

	got, err := store.IDs()
	if err != nil {
		t.Fatalf("IDs() error = %v, want nil", err)
	}

	want := []string{"alpha", "bravo", "charlie", "shared"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("IDs() = %#v, want %#v", got, want)
	}
}

func TestWorkspaceDiscoveryIDsDiscoversOnlyDirectChildDirectories(t *testing.T) {
	workspaceDirectory := t.TempDir()
	directWorkspace := filepath.Join(workspaceDirectory, "direct")
	mustMkdir(t, directWorkspace)
	mustMkdir(t, filepath.Join(directWorkspace, "nested"))

	store := NewWorkspaceDiscovery([]string{workspaceDirectory})

	got, err := store.IDs()
	if err != nil {
		t.Fatalf("IDs() error = %v, want nil", err)
	}

	want := []string{"direct"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("IDs() = %#v, want %#v", got, want)
	}
}

func TestWorkspaceDiscoveryIDsSkipsMissingWorkspaceDirectories(t *testing.T) {
	workspaceDirectory := t.TempDir()
	missingDirectory := filepath.Join(t.TempDir(), "missing")
	mustMkdir(t, filepath.Join(workspaceDirectory, "alpha"))

	store := NewWorkspaceDiscovery([]string{missingDirectory, workspaceDirectory})

	got, err := store.IDs()
	if err != nil {
		t.Fatalf("IDs() error = %v, want nil", err)
	}

	want := []string{"alpha"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("IDs() = %#v, want %#v", got, want)
	}
}

func TestWorkspaceDiscoveryResolveReturnsFirstMatchingWorkspaceDirectory(t *testing.T) {
	firstDirectory := t.TempDir()
	secondDirectory := t.TempDir()

	firstWorkspacePath := filepath.Join(firstDirectory, "shared")
	mustMkdir(t, firstWorkspacePath)
	mustMkdir(t, filepath.Join(secondDirectory, "shared"))

	store := NewWorkspaceDiscovery([]string{firstDirectory, secondDirectory})

	got, found, err := store.Resolve("shared")
	if err != nil {
		t.Fatalf("Resolve() error = %v, want nil", err)
	}
	if !found {
		t.Fatal("Resolve() found = false, want true")
	}
	if got != firstWorkspacePath {
		t.Fatalf("Resolve() = %q, want %q", got, firstWorkspacePath)
	}
}

func TestWorkspaceDiscoveryUsesCanonicalPathsForIdentity(t *testing.T) {
	firstDirectory := t.TempDir()
	secondDirectory := t.TempDir()

	workspacePath := filepath.Join(firstDirectory, "workspace")
	mustMkdir(t, workspacePath)
	workspaceAlias := filepath.Join(secondDirectory, "workspace-alias")
	mustSymlink(t, workspacePath, workspaceAlias)

	store := NewWorkspaceDiscovery([]string{firstDirectory, secondDirectory})

	workspaceIDs, err := store.IDs()
	if err != nil {
		t.Fatalf("IDs() error = %v, want nil", err)
	}

	want := []string{"workspace"}
	if !reflect.DeepEqual(workspaceIDs, want) {
		t.Fatalf("IDs() = %#v, want %#v", workspaceIDs, want)
	}

	workspaceID, found, err := store.IDForDirectory(workspaceAlias)
	if err != nil {
		t.Fatalf("IDForDirectory() error = %v, want nil", err)
	}
	if !found {
		t.Fatal("IDForDirectory() found = false, want true")
	}
	if workspaceID != "workspace" {
		t.Fatalf("IDForDirectory() = %q, want %q", workspaceID, "workspace")
	}
}

func TestWorkspaceDiscoveryIDsForDirectoriesReturnsUniqueSortedWorkspaceIDs(t *testing.T) {
	workspaceDirectory := t.TempDir()
	alphaPath := filepath.Join(workspaceDirectory, "alpha")
	bravoPath := filepath.Join(workspaceDirectory, "bravo")
	mustMkdir(t, alphaPath)
	mustMkdir(t, bravoPath)

	store := NewWorkspaceDiscovery([]string{workspaceDirectory})

	got, err := store.IDsForDirectories([]string{bravoPath, alphaPath, bravoPath, t.TempDir()})
	if err != nil {
		t.Fatalf("IDsForDirectories() error = %v, want nil", err)
	}

	want := []string{"alpha", "bravo"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("IDsForDirectories() = %#v, want %#v", got, want)
	}
}

func TestWorkspaceDiscoveryReloadReplacesWorkspaceDirectories(t *testing.T) {
	firstDirectory := t.TempDir()
	secondDirectory := t.TempDir()
	mustMkdir(t, filepath.Join(firstDirectory, "first"))
	mustMkdir(t, filepath.Join(secondDirectory, "second"))

	store := NewWorkspaceDiscovery([]string{firstDirectory})
	store.Reload([]string{secondDirectory})

	got, err := store.IDs()
	if err != nil {
		t.Fatalf("IDs() error = %v, want nil", err)
	}

	want := []string{"second"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("IDs() = %#v, want %#v", got, want)
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()

	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatalf("Mkdir(%q) error = %v, want nil", path, err)
	}
}

func mustWriteFile(t *testing.T, path string) {
	t.Helper()

	if err := os.WriteFile(path, []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v, want nil", path, err)
	}
}

func mustSymlink(t *testing.T, target string, path string) {
	t.Helper()

	if err := os.Symlink(target, path); err != nil {
		t.Fatalf("Symlink(%q, %q) error = %v, want nil", target, path, err)
	}
}
