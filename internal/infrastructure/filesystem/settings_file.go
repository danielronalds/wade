package filesystem

import (
	"errors"
	"os"
	"path/filepath"
)

// SettingsFilePath returns the WADE settings path below an already-resolved home directory.
func (FileSystem) SettingsFilePath(homeDirectory string) string {
	return filepath.Join(homeDirectory, ".config", "wade", "config.json")
}

// SettingsFileExists reports whether the WADE settings file exists.
func (files FileSystem) SettingsFileExists(homeDirectory string) (bool, error) {
	_, err := os.Stat(files.SettingsFilePath(homeDirectory))
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

// ReadSettingsFile reads the WADE settings file.
func (files FileSystem) ReadSettingsFile(homeDirectory string) ([]byte, error) {
	return os.ReadFile(files.SettingsFilePath(homeDirectory))
}

// WriteSettingsFile directly replaces the WADE settings file contents.
func (files FileSystem) WriteSettingsFile(homeDirectory string, contents []byte) error {
	path := files.SettingsFilePath(homeDirectory)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, contents, 0o644)
}

// IsExecutableFile reports whether a path identifies an executable regular file.
func (FileSystem) IsExecutableFile(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return !info.IsDir() && info.Mode().Perm()&0o111 != 0, nil
}
