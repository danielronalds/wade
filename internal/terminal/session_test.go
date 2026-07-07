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

func TestInteractiveShellStartsShellDirectly(t *testing.T) {
	command := interactiveShell("/bin/zsh")
	want := []string{"/bin/zsh"}

	if !reflect.DeepEqual(command.Args, want) {
		t.Fatalf("Args = %#v, want %#v", command.Args, want)
	}
}
