package project

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type Store struct {
	directories []string
}

func NewStore(directories []string) Store {
	return Store{directories: directories}
}

func (s Store) Path(name string) (string, error) {
	if !isValidName(name) {
		return "", errors.New("invalid project name")
	}

	for _, directory := range s.directories {
		if directory == "" {
			return "", errors.New("invalid project directory")
		}

		path := filepath.Join(directory, name)
		info, err := os.Stat(path)
		if err == nil && info.IsDir() {
			return path, nil
		}

		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
	}

	return "", fmt.Errorf("project %q not found", name)
}

func isValidName(name string) bool {
	if name == "" || name == "." || name == ".." {
		return false
	}

	if filepath.IsAbs(name) {
		return false
	}

	return filepath.Base(name) == name
}
