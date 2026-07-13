package terminalsessions

// TODO: Review properly

import (
	"reflect"
	"strings"
	"testing"
)

func TestShellCommandRunsCommandThroughShell(t *testing.T) {
	command := shellCommand("/bin/zsh", "wade", "pi -c")
	want := []string{"/bin/zsh", "-lc", "pi -c"}

	if !reflect.DeepEqual(command.Args, want) {
		t.Fatalf("Args = %#v, want %#v", command.Args, want)
	}
}

func TestShellCommandSetsShellEnvironment(t *testing.T) {
	command := shellCommand("/bin/zsh", "wade", "pi -c")

	if !hasEnvironment(command.Env, "SHELL=/bin/zsh") {
		t.Fatalf("Env does not contain SHELL=/bin/zsh: %#v", command.Env)
	}
}

func TestShellCommandSetsWadeSessionEnvironment(t *testing.T) {
	command := shellCommand("/bin/zsh", "wade", "pi -c")

	if !hasEnvironment(command.Env, "WADE_SESSION=wade") {
		t.Fatalf("Env does not contain WADE_SESSION=wade: %#v", command.Env)
	}
}

func TestInteractiveShellStartsShellDirectly(t *testing.T) {
	command := interactiveShell("/bin/zsh", "wade")
	want := []string{"/bin/zsh"}

	if !reflect.DeepEqual(command.Args, want) {
		t.Fatalf("Args = %#v, want %#v", command.Args, want)
	}
}

func TestInteractiveShellSetsShellEnvironment(t *testing.T) {
	command := interactiveShell("/bin/zsh", "wade")

	if !hasEnvironment(command.Env, "SHELL=/bin/zsh") {
		t.Fatalf("Env does not contain SHELL=/bin/zsh: %#v", command.Env)
	}
}

func TestInteractiveShellSetsWadeSessionEnvironment(t *testing.T) {
	command := interactiveShell("/bin/zsh", "wade")

	if !hasEnvironment(command.Env, "WADE_SESSION=wade") {
		t.Fatalf("Env does not contain WADE_SESSION=wade: %#v", command.Env)
	}
}

func TestShellEnvironmentReplacesInheritedWadeSession(t *testing.T) {
	t.Setenv("WADE_SESSION", "outer")

	command := interactiveShell("/bin/zsh", "wade")
	wadeSessions := environmentValues(command.Env, "WADE_SESSION")
	want := []string{"wade"}

	if !reflect.DeepEqual(wadeSessions, want) {
		t.Fatalf("WADE_SESSION values = %#v, want %#v", wadeSessions, want)
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
