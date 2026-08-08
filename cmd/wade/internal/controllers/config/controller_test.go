package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"wade/internal/models/settings"
)

type settingsModelStub struct {
	path string
	err  error
}

func (stub settingsModelStub) EnsureFile() (string, error) {
	return stub.path, stub.err
}
func (stub settingsModelStub) Get() (settings.Settings, error) {
	return settings.Settings{}, stub.err
}
func (stub settingsModelStub) LoadRuntimeConfiguration() (settings.RuntimeConfiguration, error) {
	return settings.RuntimeConfiguration{}, stub.err
}
func (stub settingsModelStub) Update(settings.Settings) (settings.UpdateResult, error) {
	return settings.UpdateResult{}, stub.err
}
func (stub settingsModelStub) Reload() (settings.UpdateResult, error) {
	return settings.UpdateResult{}, stub.err
}

func TestControllerOpensInjectedSettingsPath(t *testing.T) {
	directory := t.TempDir()
	capturedPath := filepath.Join(directory, "captured")
	editor := filepath.Join(directory, "editor")
	script := "#!/bin/sh\nprintf '%s' \"$1\" > " + capturedPath + "\n"
	if err := os.WriteFile(editor, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("EDITOR", editor)

	settingsPath := filepath.Join(directory, "config.json")
	exitCode, err := NewController(settingsModelStub{path: settingsPath}).HandleArgs([]string{"config"})
	if err != nil || exitCode != 0 {
		t.Fatalf("HandleArgs() = %d, %v", exitCode, err)
	}
	contents, err := os.ReadFile(capturedPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != settingsPath {
		t.Fatalf("editor path = %q, want %q", contents, settingsPath)
	}
}

func TestControllerReturnsSettingsErrors(t *testing.T) {
	wantError := errors.New("settings failure")
	_, err := NewController(settingsModelStub{err: wantError}).HandleArgs([]string{"config"})
	if !errors.Is(err, wantError) {
		t.Fatalf("HandleArgs() error = %v, want %v", err, wantError)
	}
}

func TestGetEditorUsesExplicitEmptyEditor(t *testing.T) {
	t.Setenv("EDITOR", "")
	if editor := getEditor(); editor != "" {
		t.Fatalf("getEditor() = %q, want explicit empty EDITOR", editor)
	}
}
