package filesystem

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSettingsFileReadWriteAndExistence(t *testing.T) {
	homeDirectory := t.TempDir()
	files := NewFileSystem()

	if path := files.SettingsFilePath(homeDirectory); path != filepath.Join(homeDirectory, ".config", "wade", "config.json") {
		t.Fatalf("SettingsFilePath() = %q", path)
	}

	exists, err := files.SettingsFileExists(homeDirectory)
	if err != nil || exists {
		t.Fatalf("SettingsFileExists() = %t, error = %v, want false, nil", exists, err)
	}

	want := []byte("{\n  \"themeAccentColor\": \"white\"\n}\n")
	if err := files.WriteSettingsFile(homeDirectory, want); err != nil {
		t.Fatalf("WriteSettingsFile() error = %v", err)
	}

	exists, err = files.SettingsFileExists(homeDirectory)
	if err != nil || !exists {
		t.Fatalf("SettingsFileExists() = %t, error = %v, want true, nil", exists, err)
	}
	got, err := files.ReadSettingsFile(homeDirectory)
	if err != nil {
		t.Fatalf("ReadSettingsFile() error = %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("ReadSettingsFile() = %q, want %q", got, want)
	}
}

func TestIsExecutableFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shell")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	executable, err := NewFileSystem().IsExecutableFile(path)
	if err != nil || !executable {
		t.Fatalf("IsExecutableFile() = %t, error = %v, want true, nil", executable, err)
	}
}
