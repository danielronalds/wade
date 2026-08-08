package repositories

import (
	"errors"
	"testing"
)

func TestValidateRepositoryID(t *testing.T) {
	tests := map[string]struct {
		repositoryID string
		wantError    bool
	}{
		"basename":      {repositoryID: "wade"},
		"empty":         {repositoryID: "", wantError: true},
		"relative path": {repositoryID: "example/wade", wantError: true},
		"absolute path": {repositoryID: "/example/wade", wantError: true},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := validateRepositoryID(test.repositoryID)
			if !test.wantError {
				if err != nil {
					t.Fatalf("validateRepositoryID() error = %v, want nil", err)
				}
				return
			}

			var invalidIDError InvalidRepositoryIDError
			if !errors.As(err, &invalidIDError) {
				t.Fatalf("validateRepositoryID() error = %v, want InvalidRepositoryIDError", err)
			}
		})
	}
}
