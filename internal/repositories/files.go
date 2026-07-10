package repositories

// TODO: Review properly

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

type FileRepository struct{}

func NewFileRepository() FileRepository {
	return FileRepository{}
}

func (r FileRepository) EnsureDirectory(path string) error {
	return os.MkdirAll(path, 0o755)
}

func (r FileRepository) EnsurePathDoesNotExist(path string) error {
	_, err := os.Stat(path)
	if err == nil {
		return os.ErrExist
	}

	if errors.Is(err, os.ErrNotExist) {
		return nil
	}

	return err
}

func (r FileRepository) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (r FileRepository) CopyPath(source string, destination string) error {
	info, err := os.Lstat(source)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if info.Mode()&os.ModeSymlink != 0 {
		return r.copySymlink(source, destination)
	}

	if info.IsDir() {
		return r.copyDirectory(source, destination)
	}

	return r.copyFile(source, destination, info.Mode())
}

func (r FileRepository) copySymlink(source string, destination string) error {
	linkTarget, err := os.Readlink(source)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}

	if err := os.RemoveAll(destination); err != nil {
		return err
	}

	return os.Symlink(linkTarget, destination)
}

func (r FileRepository) copyDirectory(source string, destination string) error {
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relPath, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}

		target := filepath.Join(destination, relPath)
		info, err := entry.Info()
		if err != nil {
			return err
		}

		if entry.Type()&os.ModeSymlink != 0 {
			return r.copySymlink(path, target)
		}

		if entry.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}

		return r.copyFile(path, target, info.Mode())
	})
}

func (r FileRepository) copyFile(source string, destination string, mode fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}

	sourceFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destinationFile, err := os.OpenFile(destination, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, sourceFile)
	return err
}
