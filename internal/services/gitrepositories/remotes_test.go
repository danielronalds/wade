package gitrepositories

import "testing"

func TestCanonicalRemoteIdentity(t *testing.T) {
	tests := map[string]struct {
		remoteURL string
		want      string
	}{
		"ssh": {
			remoteURL: "git@github.com:Example/WADE.git",
			want:      "github.com/example/wade",
		},
		"https": {
			remoteURL: "https://github.com/example/wade.git",
			want:      "github.com/example/wade",
		},
		"ssh URL": {
			remoteURL: "ssh://git@github.com/example/wade.git",
			want:      "github.com/example/wade",
		},
		"unsupported local path": {
			remoteURL: "/tmp/wade",
			want:      "",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := CanonicalRemoteIdentity(test.remoteURL)
			if got != test.want {
				t.Fatalf("CanonicalRemoteIdentity() = %q, want %q", got, test.want)
			}
		})
	}
}
