package gitrepositories

import "path/filepath"

func validateRepositoryID(repositoryID string) error {
	if repositoryID == "" || repositoryID == "." || repositoryID == ".." {
		return InvalidRepositoryIDError{RepositoryID: repositoryID}
	}
	if filepath.IsAbs(repositoryID) || filepath.Base(repositoryID) != repositoryID {
		return InvalidRepositoryIDError{RepositoryID: repositoryID}
	}

	return nil
}
