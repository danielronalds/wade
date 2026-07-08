package terminal

import (
	"reflect"
	"testing"
)

func TestShellCommandRunsCommandThroughShell(t *testing.T) {
	command := shellCommand("/bin/zsh", "pi -c")
	want := []string{"/bin/zsh", "-lc", "pi -c"}

	if !reflect.DeepEqual(command.Args, want) {
		t.Fatalf("Args = %#v, want %#v", command.Args, want)
	}
}

func TestShellCommandSetsShellEnvironment(t *testing.T) {
	command := shellCommand("/bin/zsh", "pi -c")

	if !hasEnvironment(command.Env, "SHELL=/bin/zsh") {
		t.Fatalf("Env does not contain SHELL=/bin/zsh: %#v", command.Env)
	}
}

func TestInteractiveShellStartsShellDirectly(t *testing.T) {
	command := interactiveShell("/bin/zsh")
	want := []string{"/bin/zsh"}

	if !reflect.DeepEqual(command.Args, want) {
		t.Fatalf("Args = %#v, want %#v", command.Args, want)
	}
}

func TestInteractiveShellSetsShellEnvironment(t *testing.T) {
	command := interactiveShell("/bin/zsh")

	if !hasEnvironment(command.Env, "SHELL=/bin/zsh") {
		t.Fatalf("Env does not contain SHELL=/bin/zsh: %#v", command.Env)
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
