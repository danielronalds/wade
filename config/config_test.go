package config

import "testing"

func TestLoadUsesRunAddressByDefault(t *testing.T) {
	t.Setenv(addressEnv, "")
	t.Setenv(devModeEnv, "")
	t.Setenv("SHELL", "/bin/zsh")

	configuration := Load()

	if configuration.Address != "editor.localhost:8765" {
		t.Fatalf("expected run address, got %q", configuration.Address)
	}

	if configuration.Shell != "/bin/zsh" {
		t.Fatalf("expected shell to come from environment, got %q", configuration.Shell)
	}
}

func TestLoadUsesDevAddressWhenDevModeIsEnabled(t *testing.T) {
	t.Setenv(addressEnv, "")
	t.Setenv(devModeEnv, "1")

	configuration := Load()

	if configuration.Address != "editor-dev.localhost:8765" {
		t.Fatalf("expected dev address, got %q", configuration.Address)
	}
}

func TestLoadAllowsAddressOverride(t *testing.T) {
	t.Setenv(addressEnv, "127.0.0.1:9999")
	t.Setenv(devModeEnv, "1")

	configuration := Load()

	if configuration.Address != "127.0.0.1:9999" {
		t.Fatalf("expected overridden address, got %q", configuration.Address)
	}
}

func TestLoadTreatsFalseDevModeAsRunMode(t *testing.T) {
	t.Setenv(addressEnv, "")
	t.Setenv(devModeEnv, "false")

	configuration := Load()

	if configuration.Address != "editor.localhost:8765" {
		t.Fatalf("expected run address, got %q", configuration.Address)
	}
}
