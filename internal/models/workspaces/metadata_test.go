package workspaces

import "testing"

func TestRepositoryURL(t *testing.T) {
	tests := map[string]struct {
		repositoryID string
		want         string
	}{
		"github repository": {repositoryID: "danielronalds/wade", want: "https://github.com/danielronalds/wade"},
		"empty repository":  {repositoryID: "", want: ""},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if got := repositoryURL(test.repositoryID); got != test.want {
				t.Fatalf("repositoryURL() = %q, want %q", got, test.want)
			}
		})
	}
}
