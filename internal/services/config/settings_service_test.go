package config

import (
	"reflect"
	"testing"

	"wade/internal/repositories"
)

type settingsRepositoryStub struct {
	settings  Settings
	saved     Settings
	saveCalls int
	err       error
}

func (s *settingsRepositoryStub) Load() (Settings, error) {
	return s.settings, s.err
}

func (s *settingsRepositoryStub) Save(settings Settings) error {
	s.saved = settings
	s.saveCalls++
	return s.err
}

type runtimeConfigApplierStub struct {
	configuration Config
	calls         int
}

func (s *runtimeConfigApplierStub) ApplyConfig(configuration Config) {
	s.configuration = configuration
	s.calls++
}

func TestServiceUpdateNormalisesPersistsAndAppliesSettings(t *testing.T) {
	workspaceDirectory := t.TempDir()
	repository := &settingsRepositoryStub{settings: validSettings(workspaceDirectory)}
	runtime := &runtimeConfigApplierStub{}
	service := NewService(repository, runtime)

	request := validSettings("  " + workspaceDirectory + "  ")
	request.Agents[0].Name = " Pi "
	request.Agents[0].Command = " pi -c "
	request.WorktreeCopyExcludes = []string{" node_modules ", ""}

	got, err := service.Update(request)
	if err != nil {
		t.Fatalf("Update() error = %v, want nil", err)
	}
	if !reflect.DeepEqual(got.WorkspaceDirectories, []string{workspaceDirectory}) {
		t.Fatalf("Update() WorkspaceDirectories = %#v, want [%s]", got.WorkspaceDirectories, workspaceDirectory)
	}
	if got.Agents[0].Name != "Pi" || got.Agents[0].Command != "pi -c" {
		t.Fatalf("Update() agent = %#v, want normalised Pi", got.Agents[0])
	}
	if !reflect.DeepEqual(repository.saved.WorkspaceDirectories, got.WorkspaceDirectories) {
		t.Fatalf("saved WorkspaceDirectories = %#v, want %#v", repository.saved.WorkspaceDirectories, got.WorkspaceDirectories)
	}
	if runtime.calls != 1 || !reflect.DeepEqual(runtime.configuration.WorkspaceDirs, []string{workspaceDirectory}) {
		t.Fatalf("runtime application = %d/%#v, want one call with workspace", runtime.calls, runtime.configuration.WorkspaceDirs)
	}
}

func TestServiceUpdateDoesNotPersistOrApplyInvalidSettings(t *testing.T) {
	repository := &settingsRepositoryStub{settings: validSettings(t.TempDir())}
	runtime := &runtimeConfigApplierStub{}
	service := NewService(repository, runtime)
	request := validSettings("relative/path")

	if _, err := service.Update(request); err == nil {
		t.Fatal("Update() error = nil, want validation error")
	}
	if repository.saveCalls != 0 {
		t.Fatalf("Save() calls = %d, want 0", repository.saveCalls)
	}
	if runtime.calls != 0 {
		t.Fatalf("ApplyConfig() calls = %d, want 0", runtime.calls)
	}
}

func TestServiceReloadAppliesOutOfBandSettings(t *testing.T) {
	workspaceDirectory := t.TempDir()
	repository := &settingsRepositoryStub{settings: validSettings(workspaceDirectory)}
	runtime := &runtimeConfigApplierStub{}
	service := NewService(repository, runtime)

	got, err := service.Reload()
	if err != nil {
		t.Fatalf("Reload() error = %v, want nil", err)
	}
	if runtime.calls != 1 {
		t.Fatalf("ApplyConfig() calls = %d, want 1", runtime.calls)
	}
	if !reflect.DeepEqual(got.WorkspaceDirectories, []string{workspaceDirectory}) {
		t.Fatalf("Reload() WorkspaceDirectories = %#v, want [%s]", got.WorkspaceDirectories, workspaceDirectory)
	}
}

func validSettings(workspaceDirectory string) Settings {
	return Settings{
		WorkspaceDirectories: []string{workspaceDirectory},
		Agents: []repositories.Agent{
			{Name: "Pi", Command: "pi -c", Default: true},
		},
		WorktreeCopyExcludes: []string{},
		ThemeAccentColor:     repositories.ThemeAccentColorWhite,
	}
}
