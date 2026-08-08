package repositories

import (
	"errors"
	"testing"
)

func TestValidateWorktreeID(t *testing.T) {
	tests := map[string]struct {
		worktreeID string
		wantError  bool
	}{
		"basename":      {worktreeID: "wade-feature"},
		"empty":         {worktreeID: "", wantError: true},
		"relative path": {worktreeID: "worktrees/wade-feature", wantError: true},
		"absolute path": {worktreeID: "/worktrees/wade-feature", wantError: true},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := validateWorktreeID(test.worktreeID)
			if !test.wantError {
				if err != nil {
					t.Fatalf("validateWorktreeID() error = %v, want nil", err)
				}
				return
			}

			var invalidIDError InvalidWorktreeIDError
			if !errors.As(err, &invalidIDError) {
				t.Fatalf("validateWorktreeID() error = %v, want InvalidWorktreeIDError", err)
			}
		})
	}
}
