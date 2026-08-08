package controllers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"
	"time"

	"wade/internal/models/settings"
	"wade/internal/models/workspaces"
)

func TestUpdateSettingsDecodesModelResourceAndConfiguresRuntimeModels(t *testing.T) {
	requested := settings.Settings{
		WorkspaceDirectories: []string{"~/Code"},
		Agents:               []settings.Agent{{Name: "Pi", Command: "pi -c", Default: true}},
		ThemeAccentColor:     settings.ThemeAccentColorPurple,
	}
	result := settings.UpdateResult{
		Settings: requested,
		RuntimeConfiguration: settings.RuntimeConfiguration{
			WorkspaceDirectoryPaths:            []string{"/home/test/Code"},
			WorkspaceDirectorySettings:         []string{"~/Code"},
			Shell:                              "/bin/zsh",
			Agents:                             requested.Agents,
			CopyIgnoredFilesOnWorktreeCreation: true,
			WorktreeCopyExcludes:               []string{"node_modules"},
		},
	}
	settingsModel := &fakeSettingsModel{updateResult: result}
	workspaceModel := &fakeWorkspacesModel{}
	repositoryModel := &fakeRepositoriesModel{}
	terminalModel := &fakeTerminalsModel{}
	controller := NewSettings(settingsModel, workspaceModel, repositoryModel, terminalModel)

	body, err := json.Marshal(requested)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPut, "/api/v1/settings", bytes.NewReader(body))
	response := httptest.NewRecorder()
	controller.Update(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if !reflect.DeepEqual(settingsModel.updated, requested) {
		t.Fatalf("Update() request = %#v", settingsModel.updated)
	}
	wantWorkspaceConfiguration := workspaces.Configuration{WorkspaceDirectories: []workspaces.WorkspaceDirectory{{Setting: "~/Code", Path: "/home/test/Code"}}}
	if !reflect.DeepEqual(workspaceModel.configuration, wantWorkspaceConfiguration) {
		t.Fatalf("workspace configuration = %#v", workspaceModel.configuration)
	}
	if !repositoryModel.configuration.CopyIgnoredFilesOnWorktreeCreation || !reflect.DeepEqual(repositoryModel.configuration.WorktreeCopyExcludes, []string{"node_modules"}) {
		t.Fatalf("repository configuration = %#v", repositoryModel.configuration)
	}
	if terminalModel.configuration.Shell != "/bin/zsh" || len(terminalModel.configuration.Agents) != 1 {
		t.Fatalf("terminal configuration = %#v", terminalModel.configuration)
	}
}

func TestUpdateSettingsMapsModelValidationError(t *testing.T) {
	settingsModel := &fakeSettingsModel{updateError: settings.InvalidSettingsError{Err: errors.New("invalid")}}
	controller := NewSettings(settingsModel, &fakeWorkspacesModel{}, &fakeRepositoriesModel{}, &fakeTerminalsModel{})
	request := httptest.NewRequest(http.MethodPut, "/api/v1/settings", bytes.NewBufferString(`{"workspaceDirectories":[],"shell":"","agents":[],"copyIgnoredFilesOnWorktreeCreation":false,"openWorktreesInNewTabs":false,"worktreeCopyExcludes":[],"themeAccentColor":"white"}`))
	response := httptest.NewRecorder()

	controller.Update(response, request)

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnprocessableEntity)
	}
}

type orderedSettingsModel struct {
	calls chan int
}

func (*orderedSettingsModel) EnsureFile() (string, error) {
	return "", nil
}

func (*orderedSettingsModel) Get() (settings.Settings, error) {
	return settings.Settings{}, nil
}

func (*orderedSettingsModel) LoadRuntimeConfiguration() (settings.RuntimeConfiguration, error) {
	return settings.RuntimeConfiguration{}, nil
}

func (model *orderedSettingsModel) Update(settings.Settings) (settings.UpdateResult, error) {
	model.calls <- 2
	return runtimeResult("new", "/new"), nil
}

func (model *orderedSettingsModel) Reload() (settings.UpdateResult, error) {
	model.calls <- 1
	return runtimeResult("old", "/old"), nil
}

type blockingWorkspacesModel struct {
	*fakeWorkspacesModel
	firstConfigurationStarted chan struct{}
	releaseFirstConfiguration chan struct{}

	mu             sync.Mutex
	configurations []workspaces.Configuration
}

func (model *blockingWorkspacesModel) Configure(configuration workspaces.Configuration) {
	model.mu.Lock()
	call := len(model.configurations)
	model.configurations = append(model.configurations, configuration)
	model.mu.Unlock()
	if call == 0 {
		close(model.firstConfigurationStarted)
		<-model.releaseFirstConfiguration
	}
}

func TestSettingsControllerSerialisesPersistenceAndRuntimeOrchestration(t *testing.T) {
	settingsModel := &orderedSettingsModel{calls: make(chan int, 2)}
	workspaceModel := &blockingWorkspacesModel{
		fakeWorkspacesModel:       &fakeWorkspacesModel{},
		firstConfigurationStarted: make(chan struct{}),
		releaseFirstConfiguration: make(chan struct{}),
	}
	controller := NewSettings(settingsModel, workspaceModel, &fakeRepositoriesModel{}, &fakeTerminalsModel{})

	firstDone := make(chan struct{})
	go func() {
		defer close(firstDone)
		controller.Reload(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/v1/settings/reload", nil))
	}()
	<-workspaceModel.firstConfigurationStarted

	secondDone := make(chan struct{})
	go func() {
		defer close(secondDone)
		controller.Update(httptest.NewRecorder(), settingsUpdateRequest())
	}()

	select {
	case call := <-settingsModel.calls:
		if call != 1 {
			t.Fatalf("first settings call = %d", call)
		}
	default:
		t.Fatal("first settings update was not recorded")
	}
	select {
	case call := <-settingsModel.calls:
		t.Fatalf("settings call %d started before earlier runtime configuration completed", call)
	case <-time.After(30 * time.Millisecond):
	}

	close(workspaceModel.releaseFirstConfiguration)
	<-firstDone
	<-secondDone

	workspaceModel.mu.Lock()
	configurations := append([]workspaces.Configuration(nil), workspaceModel.configurations...)
	workspaceModel.mu.Unlock()
	if len(configurations) != 2 || configurations[1].WorkspaceDirectories[0].Setting != "new" {
		t.Fatalf("runtime configuration order = %#v", configurations)
	}
}

func settingsUpdateRequest() *http.Request {
	return httptest.NewRequest(http.MethodPut, "/api/v1/settings", bytes.NewBufferString(`{"workspaceDirectories":[],"shell":"","agents":[],"copyIgnoredFilesOnWorktreeCreation":false,"openWorktreesInNewTabs":false,"worktreeCopyExcludes":[],"themeAccentColor":"white"}`))
}

func runtimeResult(setting string, path string) settings.UpdateResult {
	return settings.UpdateResult{RuntimeConfiguration: settings.RuntimeConfiguration{
		WorkspaceDirectoryPaths:    []string{path},
		WorkspaceDirectorySettings: []string{setting},
	}}
}
