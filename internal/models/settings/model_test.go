package settings

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"
)

type settingsFileSystemStub struct {
	mu               sync.Mutex
	contents         []byte
	exists           bool
	executables      map[string]bool
	activeWrites     int
	maximumWrites    int
	writeDelay       time.Duration
	settingsFilePath string
}

func (stub *settingsFileSystemStub) SettingsFilePath(string) string {
	if stub.settingsFilePath != "" {
		return stub.settingsFilePath
	}
	return "/home/test/.config/wade/config.json"
}

func (stub *settingsFileSystemStub) SettingsFileExists(string) (bool, error) {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	return stub.exists, nil
}

func (stub *settingsFileSystemStub) ReadSettingsFile(string) ([]byte, error) {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	if !stub.exists {
		return nil, os.ErrNotExist
	}
	return append([]byte(nil), stub.contents...), nil
}

func (stub *settingsFileSystemStub) WriteSettingsFile(_ string, contents []byte) error {
	stub.mu.Lock()
	stub.activeWrites++
	if stub.activeWrites > stub.maximumWrites {
		stub.maximumWrites = stub.activeWrites
	}
	stub.mu.Unlock()

	time.Sleep(stub.writeDelay)

	stub.mu.Lock()
	stub.contents = append([]byte(nil), contents...)
	stub.exists = true
	stub.activeWrites--
	stub.mu.Unlock()
	return nil
}

func (stub *settingsFileSystemStub) IsExecutableFile(path string) (bool, error) {
	return stub.executables[path], nil
}

type settingsEnvironmentStub struct {
	homeDirectory string
	variables     map[string]string
	shell         string
	executables   map[string]string
	homeError     error
}

func (stub settingsEnvironmentStub) HomeDirectory() (string, error) {
	return stub.homeDirectory, stub.homeError
}

func (stub settingsEnvironmentStub) Variable(name string) string {
	return stub.variables[name]
}

func (stub settingsEnvironmentStub) InheritedShell() string {
	return stub.shell
}

func (stub settingsEnvironmentStub) LookPath(name string) (string, error) {
	path, found := stub.executables[name]
	if !found {
		return "", errors.New("not found")
	}
	return path, nil
}

func TestEnsureFileCreatesDefaultsWithoutParsingExistingFiles(t *testing.T) {
	files := &settingsFileSystemStub{settingsFilePath: "/test/config.json"}
	model := New(files, testEnvironment())

	path, err := model.EnsureFile()
	if err != nil {
		t.Fatalf("EnsureFile() error = %v", err)
	}
	if path != "/test/config.json" {
		t.Fatalf("EnsureFile() = %q", path)
	}

	settings, err := model.Get()
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !reflect.DeepEqual(settings.Agents, defaultAgents) || !reflect.DeepEqual(settings.WorkspaceDirectories, []string{"~/Personal", "~/Work"}) {
		t.Fatalf("default settings = %#v", settings)
	}

	files.mu.Lock()
	files.contents = []byte(`{`)
	files.mu.Unlock()
	if _, err := model.EnsureFile(); err != nil {
		t.Fatalf("EnsureFile() parsed existing file: %v", err)
	}
}

func TestGetReadsLegacyDirectoriesAndReturnsDetachedSettings(t *testing.T) {
	files := newSettingsFileSystem(`{"projectDirectories":["~/Legacy"],"agents":[{"name":"Custom","command":"custom","default":true}]}`)
	model := New(files, testEnvironment())

	first, err := model.Get()
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	first.WorkspaceDirectories[0] = "changed"
	first.Agents[0].Name = "changed"

	second, err := model.Get()
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if second.WorkspaceDirectories[0] != "~/Legacy" || second.Agents[0].Name != "Custom" {
		t.Fatalf("Get() returned attached settings: %#v", second)
	}
}

func TestUpdateNormalisesPreservesUnknownKeysAndResolvesRuntimeConfiguration(t *testing.T) {
	files := newSettingsFileSystem(`{"workspaceDirectories":["/old"],"agents":[{"name":"Old","command":"old","default":true}],"theme":"dark","projectDirectories":["~/Legacy"],"agentCommand":"legacy"}`)
	environment := testEnvironment()
	environment.variables[addressEnvironmentVariable] = "custom.localhost:9000"
	model := New(files, environment)

	request := validSettings("  ~/Code  ")
	request.Agents[0].Name = " Pi "
	request.Agents[0].Command = " pi -c "
	request.WorktreeCopyExcludes = []string{" node_modules ", ""}
	result, err := model.Update(request)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if !reflect.DeepEqual(result.Settings.WorkspaceDirectories, []string{"~/Code"}) {
		t.Fatalf("WorkspaceDirectories = %#v", result.Settings.WorkspaceDirectories)
	}
	if result.Settings.Agents[0].Name != "Pi" || result.Settings.Agents[0].Command != "pi -c" {
		t.Fatalf("Agent = %#v", result.Settings.Agents[0])
	}
	if !reflect.DeepEqual(result.RuntimeConfiguration.WorkspaceDirectoryPaths, []string{"/home/test/Code"}) {
		t.Fatalf("WorkspaceDirectoryPaths = %#v", result.RuntimeConfiguration.WorkspaceDirectoryPaths)
	}
	if result.RuntimeConfiguration.Address != "custom.localhost:9000" {
		t.Fatalf("Address = %q", result.RuntimeConfiguration.Address)
	}

	var persisted map[string]any
	files.mu.Lock()
	contents := append([]byte(nil), files.contents...)
	files.mu.Unlock()
	if err := json.Unmarshal(contents, &persisted); err != nil {
		t.Fatalf("decoding persisted settings: %v", err)
	}
	if persisted["theme"] != "dark" {
		t.Fatalf("unknown theme = %#v", persisted["theme"])
	}
	for _, removed := range []string{"projectDirectories", "agentCommand", "agentPaneCommand"} {
		if _, found := persisted[removed]; found {
			t.Fatalf("legacy setting %q was preserved", removed)
		}
	}
}

func TestUpdateRejectsInvalidSettingsWithoutWriting(t *testing.T) {
	files := newSettingsFileSystem(`{"workspaceDirectories":[],"agents":[{"name":"Pi","command":"pi -c","default":true}]}`)
	model := New(files, testEnvironment())
	request := validSettings("relative/path")

	_, err := model.Update(request)
	var invalid InvalidSettingsError
	if !errors.As(err, &invalid) {
		t.Fatalf("Update() error = %v, want InvalidSettingsError", err)
	}

	files.mu.Lock()
	contents := string(files.contents)
	files.mu.Unlock()
	if contents != `{"workspaceDirectories":[],"agents":[{"name":"Pi","command":"pi -c","default":true}]}` {
		t.Fatalf("invalid update wrote settings: %s", contents)
	}
}

func TestReloadNormalisesOutOfBandSettingsWithoutPersisting(t *testing.T) {
	contents := `{"workspaceDirectories":["  ~/Code  "],"agents":[{"name":" Pi ","command":" pi -c ","default":true}],"worktreeCopyExcludes":[" node_modules ",""],"themeAccentColor":""}`
	files := newSettingsFileSystem(contents)
	model := New(files, testEnvironment())

	result, err := model.Reload()
	if err != nil {
		t.Fatalf("Reload() error = %v", err)
	}
	if result.Settings.WorkspaceDirectories[0] != "~/Code" || result.Settings.ThemeAccentColor != ThemeAccentColorWhite {
		t.Fatalf("Reload() settings = %#v", result.Settings)
	}

	files.mu.Lock()
	persisted := string(files.contents)
	files.mu.Unlock()
	if persisted != contents {
		t.Fatalf("Reload() persisted normalised settings: %s", persisted)
	}
}

func TestLoadRuntimeConfigurationPreservesEnvironmentPrecedence(t *testing.T) {
	files := newSettingsFileSystem(`{"workspaceDirectories":["~/Code"],"agents":[{"name":"Pi","command":"pi -c","default":true}]}`)
	environment := testEnvironment()
	environment.variables[addressEnvironmentVariable] = "inherited.localhost:9000"
	environment.variables[developmentEnvironment] = "1"
	environment.shell = "/bin/zsh"
	model := New(files, environment)

	configuration, err := model.LoadRuntimeConfiguration()
	if err != nil {
		t.Fatalf("LoadRuntimeConfiguration() error = %v", err)
	}
	if configuration.Address != "editor-dev.localhost:8090" {
		t.Fatalf("Address = %q", configuration.Address)
	}
	if configuration.Shell != "/bin/zsh" {
		t.Fatalf("Shell = %q", configuration.Shell)
	}
	if !reflect.DeepEqual(configuration.WorkspaceDirectorySettings, []string{"~/Code"}) {
		t.Fatalf("WorkspaceDirectorySettings = %#v", configuration.WorkspaceDirectorySettings)
	}
}

func TestLoadRuntimeConfigurationResolvesConfiguredShell(t *testing.T) {
	files := newSettingsFileSystem(`{"workspaceDirectories":[],"shell":"custom-shell","agents":[{"name":"Pi","command":"pi -c","default":true}]}`)
	environment := testEnvironment()
	environment.executables["custom-shell"] = "/test/bin/custom-shell"

	configuration, err := New(files, environment).LoadRuntimeConfiguration()
	if err != nil {
		t.Fatalf("LoadRuntimeConfiguration() error = %v", err)
	}
	if configuration.Shell != "/test/bin/custom-shell" {
		t.Fatalf("Shell = %q", configuration.Shell)
	}
}

func TestUpdateSerialisesSettingsWrites(t *testing.T) {
	files := newSettingsFileSystem(`{"workspaceDirectories":[],"agents":[{"name":"Pi","command":"pi -c","default":true}]}`)
	files.writeDelay = 5 * time.Millisecond
	model := New(files, testEnvironment())

	var wait sync.WaitGroup
	for range 8 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			if _, err := model.Update(validSettings("~/Code")); err != nil {
				t.Errorf("Update() error = %v", err)
			}
		}()
	}
	wait.Wait()

	files.mu.Lock()
	maximumWrites := files.maximumWrites
	files.mu.Unlock()
	if maximumWrites != 1 {
		t.Fatalf("maximum concurrent writes = %d, want 1", maximumWrites)
	}
}

func newSettingsFileSystem(contents string) *settingsFileSystemStub {
	return &settingsFileSystemStub{contents: []byte(contents), exists: true, executables: make(map[string]bool)}
}

func testEnvironment() settingsEnvironmentStub {
	return settingsEnvironmentStub{
		homeDirectory: "/home/test",
		variables:     make(map[string]string),
		shell:         "/bin/sh",
		executables:   make(map[string]string),
	}
}

func validSettings(workspaceDirectory string) Settings {
	return Settings{
		WorkspaceDirectories: []string{workspaceDirectory},
		Agents: []Agent{
			{Name: "Pi", Command: "pi -c", Default: true},
		},
		WorktreeCopyExcludes: []string{},
		ThemeAccentColor:     ThemeAccentColorWhite,
	}
}
