package settings

import (
	"errors"
	"testing"
)

func TestNormaliseAndValidateSettingsAcceptsSupportedShellForms(t *testing.T) {
	files := &settingsFileSystemStub{executables: map[string]bool{"/test/custom-shell": true}}
	environment := testEnvironment()
	environment.executables["custom-shell"] = "/path/custom-shell"

	for name, shell := range map[string]string{
		"empty":               "",
		"absolute executable": "/test/custom-shell",
		"command on PATH":     "custom-shell",
	} {
		t.Run(name, func(t *testing.T) {
			request := validSettings("~/Code")
			request.Shell = shell
			if _, err := normaliseAndValidateSettings(request, environment.homeDirectory, files, environment); err != nil {
				t.Fatalf("normaliseAndValidateSettings() error = %v", err)
			}
		})
	}
}

func TestNormaliseAndValidateSettingsRejectsInvalidShells(t *testing.T) {
	files := &settingsFileSystemStub{executables: make(map[string]bool)}
	environment := testEnvironment()

	for name, shell := range map[string]string{
		"arguments":        "/bin/zsh -l",
		"missing absolute": "/missing/shell",
		"missing command":  "missing-shell",
		"relative path":    "bin/shell",
	} {
		t.Run(name, func(t *testing.T) {
			request := validSettings("~/Code")
			request.Shell = shell
			if _, err := normaliseAndValidateSettings(request, environment.homeDirectory, files, environment); err == nil {
				t.Fatal("normaliseAndValidateSettings() error = nil")
			}
		})
	}
}

func TestValidateAgentsRejectsInvalidConfigurations(t *testing.T) {
	tests := map[string][]Agent{
		"missing agents": nil,
		"empty name": {
			{Name: " ", Command: "pi -c", Default: true},
		},
		"empty command": {
			{Name: "Pi", Command: " ", Default: true},
		},
		"case-insensitive duplicate": {
			{Name: "Pi", Command: "pi -c", Default: true},
			{Name: "pi", Command: "pi -c"},
		},
		"missing default": {
			{Name: "Pi", Command: "pi -c"},
		},
		"multiple defaults": {
			{Name: "Pi", Command: "pi -c", Default: true},
			{Name: "Claude", Command: "claude", Default: true},
		},
	}

	for name, agents := range tests {
		t.Run(name, func(t *testing.T) {
			if err := validateAgents(agents); err == nil {
				t.Fatal("validateAgents() error = nil")
			}
		})
	}
}

func TestNormaliseAndValidateSettingsRejectsInvalidDomainValues(t *testing.T) {
	files := &settingsFileSystemStub{executables: make(map[string]bool)}
	environment := testEnvironment()

	tests := map[string]func(*Settings){
		"relative workspace directory": func(settings *Settings) {
			settings.WorkspaceDirectories = []string{"relative/path"}
		},
		"invalid worktree exclude": func(settings *Settings) {
			settings.WorktreeCopyExcludes = []string{"["}
		},
		"invalid theme accent": func(settings *Settings) {
			settings.ThemeAccentColor = "blue"
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			request := validSettings("~/Code")
			mutate(&request)
			if _, err := normaliseAndValidateSettings(request, environment.homeDirectory, files, environment); err == nil {
				t.Fatal("normaliseAndValidateSettings() error = nil")
			}
		})
	}
}

func TestNormaliseAndValidateSettingsTrimsValuesAndDefaultsTheme(t *testing.T) {
	request := validSettings("  ~/Code  ")
	request.Agents[0].Name = " Pi "
	request.Agents[0].Command = " pi -c "
	request.WorktreeCopyExcludes = []string{" node_modules ", ""}
	request.ThemeAccentColor = ""

	normalised, err := normaliseAndValidateSettings(request, "/home/test", &settingsFileSystemStub{}, testEnvironment())
	if err != nil {
		t.Fatalf("normaliseAndValidateSettings() error = %v", err)
	}
	if normalised.WorkspaceDirectories[0] != "~/Code" || normalised.Agents[0].Name != "Pi" || normalised.Agents[0].Command != "pi -c" {
		t.Fatalf("normalised settings = %#v", normalised)
	}
	if len(normalised.WorktreeCopyExcludes) != 1 || normalised.WorktreeCopyExcludes[0] != "node_modules" {
		t.Fatalf("WorktreeCopyExcludes = %#v", normalised.WorktreeCopyExcludes)
	}
	if normalised.ThemeAccentColor != ThemeAccentColorWhite {
		t.Fatalf("ThemeAccentColor = %q", normalised.ThemeAccentColor)
	}
}

func TestInvalidSettingsErrorUnwrapsCause(t *testing.T) {
	cause := errors.New("invalid")
	if !errors.Is(InvalidSettingsError{Err: cause}, cause) {
		t.Fatal("InvalidSettingsError does not unwrap its cause")
	}
}
