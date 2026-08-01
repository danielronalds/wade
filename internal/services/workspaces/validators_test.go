package workspaces

import (
	"errors"
	"testing"
)

func TestValidateWorkspaceID(t *testing.T) {
	tests := map[string]struct {
		workspaceID string
		wantError   bool
	}{
		"basename": {
			workspaceID: "wade",
		},
		"basename with spaces": {
			workspaceID: "wade feature",
		},
		"empty": {
			workspaceID: "",
			wantError:   true,
		},
		"current directory": {
			workspaceID: ".",
			wantError:   true,
		},
		"parent directory": {
			workspaceID: "..",
			wantError:   true,
		},
		"relative path": {
			workspaceID: "projects/wade",
			wantError:   true,
		},
		"absolute path": {
			workspaceID: "/projects/wade",
			wantError:   true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := validateWorkspaceID(test.workspaceID)
			if !test.wantError {
				if err != nil {
					t.Fatalf("validateWorkspaceID() error = %v, want nil", err)
				}
				return
			}

			var invalidIDError InvalidWorkspaceIDError
			if !errors.As(err, &invalidIDError) {
				t.Fatalf("validateWorkspaceID() error = %v, want InvalidWorkspaceIDError", err)
			}
		})
	}
}
