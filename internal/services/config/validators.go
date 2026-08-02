package config

// TODO: Review properly

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidateWorkspaceDirectories checks that workspace directory settings are usable.
func ValidateWorkspaceDirectories(directories []string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home directory: %w", err)
	}

	_, err = resolveWorkspaceDirectories(homeDir, directories)
	return err
}

// resolveWorkspaceDirectories expands configured workspace directory paths.
func resolveWorkspaceDirectories(homeDir string, directories []string) ([]string, error) {
	workspaceDirs := make([]string, 0, len(directories))
	for _, directory := range directories {
		projectDir, err := resolveWorkspaceDirectory(homeDir, directory)
		if err != nil {
			return nil, err
		}

		workspaceDirs = append(workspaceDirs, projectDir)
	}

	return workspaceDirs, nil
}

// resolveWorkspaceDirectory expands a single path using ~ or an absolute path.
func resolveWorkspaceDirectory(homeDir string, directory string) (string, error) {
	if directory == "" {
		return "", errors.New("workspace directory cannot be empty")
	}

	if directory == "~" {
		return homeDir, nil
	}

	if strings.HasPrefix(directory, "~/") {
		return filepath.Join(homeDir, strings.TrimPrefix(directory, "~/")), nil
	}

	if filepath.IsAbs(directory) {
		return filepath.Clean(directory), nil
	}

	return "", fmt.Errorf("workspace directory %q must use ~ or an absolute path", directory)
}
