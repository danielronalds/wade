package repositories

// TODO: Review properly

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestStoreNamesReturnsUniqueSortedProjectNames(t *testing.T) {
	firstDirectory := t.TempDir()
	secondDirectory := t.TempDir()

	mustMkdir(t, filepath.Join(firstDirectory, "bravo"))
	mustMkdir(t, filepath.Join(firstDirectory, "alpha"))
	mustMkdir(t, filepath.Join(firstDirectory, "shared"))
	mustMkdir(t, filepath.Join(secondDirectory, "charlie"))
	mustMkdir(t, filepath.Join(secondDirectory, "shared"))
	mustWriteFile(t, filepath.Join(firstDirectory, "not-a-project"))

	store := NewStore([]string{firstDirectory, secondDirectory})

	got, err := store.Names()
	if err != nil {
		t.Fatalf("Names() error = %v, want nil", err)
	}

	want := []string{"alpha", "bravo", "charlie", "shared"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Names() = %#v, want %#v", got, want)
	}
}

func TestStoreNamesSkipsMissingProjectDirectories(t *testing.T) {
	projectDirectory := t.TempDir()
	missingDirectory := filepath.Join(t.TempDir(), "missing")
	mustMkdir(t, filepath.Join(projectDirectory, "alpha"))

	store := NewStore([]string{missingDirectory, projectDirectory})

	got, err := store.Names()
	if err != nil {
		t.Fatalf("Names() error = %v, want nil", err)
	}

	want := []string{"alpha"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Names() = %#v, want %#v", got, want)
	}
}

func TestStorePathReturnsFirstMatchingProjectDirectory(t *testing.T) {
	firstDirectory := t.TempDir()
	secondDirectory := t.TempDir()

	firstProjectPath := filepath.Join(firstDirectory, "shared")
	mustMkdir(t, firstProjectPath)
	mustMkdir(t, filepath.Join(secondDirectory, "shared"))

	store := NewStore([]string{firstDirectory, secondDirectory})

	got, err := store.Path("shared")
	if err != nil {
		t.Fatalf("Path() error = %v, want nil", err)
	}

	if got != firstProjectPath {
		t.Fatalf("Path() = %q, want %q", got, firstProjectPath)
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
