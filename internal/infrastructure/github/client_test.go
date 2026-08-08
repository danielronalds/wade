package github

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestPullRequestReturnsParsedMetadata(t *testing.T) {
	var requestedDirectory string
	var requestedName string
	var requestedArguments []string
	client := NewClient(nil)
	client.directoryRunner = func(_ context.Context, directory string, name string, arguments ...string) (string, error) {
		requestedDirectory = directory
		requestedName = name
		requestedArguments = append([]string(nil), arguments...)
		return `{"number":12,"url":"https://github.com/example/wade/pull/12","state":"OPEN","baseRefName":"main","headRefName":"feature/review"}`, nil
	}

	pullRequest, err := client.PullRequest(context.Background(), "/workspace", "feature/review")
	if err != nil {
		t.Fatalf("PullRequest() error = %v, want nil", err)
	}
	if pullRequest.Number != 12 || pullRequest.BaseRefName != "main" || pullRequest.HeadRefName != "feature/review" {
		t.Fatalf("PullRequest() = %#v", pullRequest)
	}
	if requestedDirectory != "/workspace" || requestedName != "gh" {
		t.Fatalf("directory/name = %q/%q", requestedDirectory, requestedName)
	}
	wantArguments := []string{"pr", "view", "feature/review", "--json", "number,url,state,baseRefName,headRefName"}
	if !reflect.DeepEqual(requestedArguments, wantArguments) {
		t.Fatalf("arguments = %#v, want %#v", requestedArguments, wantArguments)
	}
}

func TestPullRequestReportsProviderAndParsingFailures(t *testing.T) {
	tests := map[string]struct {
		output string
		err    error
	}{
		"provider": {err: errors.New("provider unavailable")},
		"parsing":  {output: "{"},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			client := NewClient(nil)
			client.directoryRunner = func(context.Context, string, string, ...string) (string, error) {
				return test.output, test.err
			}

			if _, err := client.PullRequest(context.Background(), "/workspace", "main"); err == nil {
				t.Fatal("PullRequest() error = nil, want error")
			}
		})
	}
}
