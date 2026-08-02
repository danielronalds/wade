package workspaces

// TODO: Review properly

import "testing"

func TestIssueReference(t *testing.T) {
	tests := map[string]struct {
		branch string
		want   *IssueReference
	}{
		"branch with ticket": {
			branch: "feature/path-123-add-topbar",
			want: &IssueReference{
				Provider: "linear",
				Key:      "PATH-123",
				URL:      "https://linear.app/signinsolutions/issue/PATH-123",
			},
		},
		"branch without ticket": {
			branch: "main",
			want:   nil,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := issueReference(test.branch)
			if test.want == nil {
				if got != nil {
					t.Fatalf("issueReference() = %#v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatalf("issueReference() = nil, want %#v", test.want)
			}
			if *got != *test.want {
				t.Fatalf("issueReference() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestRepositoryURL(t *testing.T) {
	tests := map[string]struct {
		repositoryID string
		want         string
	}{
		"github repository": {
			repositoryID: "danielronalds/wade",
			want:         "https://github.com/danielronalds/wade",
		},
		"empty repository": {
			repositoryID: "",
			want:         "",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := repositoryURL(test.repositoryID)
			if got != test.want {
				t.Fatalf("repositoryURL() = %q, want %q", got, test.want)
			}
		})
	}
}
