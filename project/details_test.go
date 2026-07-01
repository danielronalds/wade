package project

import "testing"

func TestLinearTicketURL(t *testing.T) {
	tests := map[string]struct {
		branch string
		want   string
	}{
		"branch with ticket": {
			branch: "feature/path-123-add-topbar",
			want:   "https://linear.app/signinsolutions/issue/PATH-123",
		},
		"branch without ticket": {
			branch: "main",
			want:   "",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := linearTicketURL(test.branch)
			if got != test.want {
				t.Fatalf("got %q, want %q", got, test.want)
			}
		})
	}
}

func TestGitHubURL(t *testing.T) {
	tests := map[string]struct {
		repo string
		want string
	}{
		"github repo": {
			repo: "signinsolutions/web-terminal",
			want: "https://github.com/signinsolutions/web-terminal",
		},
		"enterprise repo": {
			repo: "git.example.com/signinsolutions/web-terminal",
			want: "https://git.example.com/signinsolutions/web-terminal",
		},
		"empty repo": {
			repo: "",
			want: "",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := githubURL(test.repo)
			if got != test.want {
				t.Fatalf("got %q, want %q", got, test.want)
			}
		})
	}
}

func TestParseGitHubRepo(t *testing.T) {
	tests := map[string]struct {
		remoteURL string
		want      string
	}{
		"github ssh": {
			remoteURL: "git@github.com:signinsolutions/web-terminal.git",
			want:      "signinsolutions/web-terminal",
		},
		"github https": {
			remoteURL: "https://github.com/signinsolutions/web-terminal.git",
			want:      "signinsolutions/web-terminal",
		},
		"github ssh url": {
			remoteURL: "ssh://git@github.com/signinsolutions/web-terminal.git",
			want:      "signinsolutions/web-terminal",
		},
		"enterprise ssh": {
			remoteURL: "git@git.example.com:signinsolutions/web-terminal.git",
			want:      "git.example.com/signinsolutions/web-terminal",
		},
		"unsupported remote": {
			remoteURL: "file:///tmp/web-terminal",
			want:      "",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := parseGitHubRepo(test.remoteURL)
			if got != test.want {
				t.Fatalf("got %q, want %q", got, test.want)
			}
		})
	}
}
