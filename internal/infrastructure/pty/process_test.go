package pty

import (
	"reflect"
	"strings"
	"testing"
)

func TestShellCommandRunsCommandThroughShell(t *testing.T) {
	command := shellCommand("/bin/zsh", testWadeEnvironment(), "pi -c")
	want := []string{"/bin/zsh", "-lc", "pi -c"}

	if !reflect.DeepEqual(command.Args, want) {
		t.Fatalf("Args = %#v, want %#v", command.Args, want)
	}
}

func TestShellCommandSetsShellEnvironment(t *testing.T) {
	command := shellCommand("/bin/zsh", testWadeEnvironment(), "pi -c")

	if !hasEnvironment(command.Env, "SHELL=/bin/zsh") {
		t.Fatalf("Env does not contain SHELL=/bin/zsh: %#v", command.Env)
	}
}

func TestShellCommandSetsWorkspaceAndTerminalEnvironment(t *testing.T) {
	command := shellCommand("/bin/zsh", testWadeEnvironment(), "pi -c")

	for _, value := range []string{
		"WADE_WORKSPACE_ID=wade",
		"WADE_TERMINAL_ID=agent:pi",
	} {
		if !hasEnvironment(command.Env, value) {
			t.Fatalf("Env does not contain %s: %#v", value, command.Env)
		}
	}
}

func TestShellCommandSetsWadeAddressEnvironment(t *testing.T) {
	command := shellCommand("/bin/zsh", testWadeEnvironment(), "pi -c")

	if !hasEnvironment(command.Env, "WADE_ADDR=editor.localhost:8765") {
		t.Fatalf("Env does not contain WADE_ADDR=editor.localhost:8765: %#v", command.Env)
	}
}

func TestInteractiveShellStartsShellDirectly(t *testing.T) {
	command := interactiveShell("/bin/zsh", testWadeEnvironment())
	want := []string{"/bin/zsh"}

	if !reflect.DeepEqual(command.Args, want) {
		t.Fatalf("Args = %#v, want %#v", command.Args, want)
	}
}

func TestInteractiveShellSetsShellEnvironment(t *testing.T) {
	command := interactiveShell("/bin/zsh", testWadeEnvironment())

	if !hasEnvironment(command.Env, "SHELL=/bin/zsh") {
		t.Fatalf("Env does not contain SHELL=/bin/zsh: %#v", command.Env)
	}
}

func TestInteractiveShellSetsWadeAddressEnvironment(t *testing.T) {
	command := interactiveShell("/bin/zsh", testWadeEnvironment())

	if !hasEnvironment(command.Env, "WADE_ADDR=editor.localhost:8765") {
		t.Fatalf("Env does not contain WADE_ADDR=editor.localhost:8765: %#v", command.Env)
	}
}

func TestShellEnvironmentRemovesInheritedLegacyWadeSession(t *testing.T) {
	t.Setenv("WADE_SESSION", "outer")

	command := interactiveShell("/bin/zsh", testWadeEnvironment())
	if values := environmentValues(command.Env, "WADE_SESSION"); len(values) != 0 {
		t.Fatalf("WADE_SESSION values = %#v, want none", values)
	}
}

func TestShellEnvironmentReplacesInheritedWadeAddress(t *testing.T) {
	t.Setenv("WADE_ADDR", "outer.localhost:8765")

	command := interactiveShell("/bin/zsh", testWadeEnvironment())
	wadeAddresses := environmentValues(command.Env, "WADE_ADDR")
	want := []string{"editor.localhost:8765"}

	if !reflect.DeepEqual(wadeAddresses, want) {
		t.Fatalf("WADE_ADDR values = %#v, want %#v", wadeAddresses, want)
	}
}

func testWadeEnvironment() WadeEnvironment {
	return WadeEnvironment{
		WorkspaceID: "wade",
		TerminalID:  "agent:pi",
		Address:     "editor.localhost:8765",
	}
}

func hasEnvironment(environment []string, value string) bool {
	for _, entry := range environment {
		if entry == value {
			return true
		}
	}

	return false
}

func environmentValues(environment []string, name string) []string {
	prefix := name + "="
	values := make([]string, 0)
	for _, entry := range environment {
		if strings.HasPrefix(entry, prefix) {
			values = append(values, strings.TrimPrefix(entry, prefix))
		}
	}

	return values
}
