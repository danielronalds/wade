package environment

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClientReadsProcessEnvironment(t *testing.T) {
	homeDirectory := t.TempDir()
	executableDirectory := t.TempDir()
	executablePath := filepath.Join(executableDirectory, "custom-shell")
	if err := os.WriteFile(executablePath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", homeDirectory)
	t.Setenv("SHELL", "/bin/zsh")
	t.Setenv("WADE_TEST_VALUE", "configured")
	t.Setenv("PATH", executableDirectory)

	client := NewClient()
	gotHomeDirectory, err := client.HomeDirectory()
	if err != nil || gotHomeDirectory != homeDirectory {
		t.Fatalf("HomeDirectory() = %q, %v", gotHomeDirectory, err)
	}
	if shell := client.InheritedShell(); shell != "/bin/zsh" {
		t.Fatalf("InheritedShell() = %q", shell)
	}
	if value := client.Variable("WADE_TEST_VALUE"); value != "configured" {
		t.Fatalf("Variable() = %q", value)
	}
	if path, err := client.LookPath("custom-shell"); err != nil || path != executablePath {
		t.Fatalf("LookPath() = %q, %v", path, err)
	}
}
