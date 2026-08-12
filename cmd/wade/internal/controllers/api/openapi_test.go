// This file is completely vibecoded and needs to be properly reviewed.

package api

import (
	"slices"
	"strings"
	"testing"

	"wade/internal/openapi"
)

func TestParseOperationsParsesEmbeddedSpecification(t *testing.T) {
	operations, err := parseOperations(openapi.JSON())
	if err != nil {
		t.Fatalf("parseOperations() error = %v, want nil", err)
	}
	if len(operations) != 22 {
		t.Fatalf("parseOperations() returned %d operations, want 22", len(operations))
	}

	commands := make([]string, 0, len(operations))
	for _, operation := range operations {
		commands = append(commands, operation.Command)
	}
	if !slices.IsSorted(commands) {
		t.Fatalf("commands are not sorted: %v", commands)
	}
	for _, excluded := range []string{"connect-workspace-terminal", "get-open-api-spec"} {
		if slices.Contains(commands, excluded) {
			t.Fatalf("commands include excluded operation %s", excluded)
		}
	}

	operation, found := findOperation(operations, "send-workspace-terminal-input")
	if !found {
		t.Fatalf("send-workspace-terminal-input not found in %v", commands)
	}
	if operation.Method != "POST" {
		t.Fatalf("method = %s, want POST", operation.Method)
	}
	if operation.Path != "/api/v1/workspaces/{workspaceId}/terminals/{terminalId}/input" {
		t.Fatalf("path = %s, want the terminal input path", operation.Path)
	}
	if !operation.HasBody || !operation.BodyRequired {
		t.Fatalf("HasBody = %t, BodyRequired = %t, want both true", operation.HasBody, operation.BodyRequired)
	}
	if len(operation.Parameters) != 2 || operation.Parameters[0].Flag != "workspace-id" || operation.Parameters[1].Flag != "terminal-id" {
		t.Fatalf("parameters = %+v, want workspace-id and terminal-id", operation.Parameters)
	}
	for _, parameter := range operation.Parameters {
		if parameter.In != "path" || !parameter.Required {
			t.Fatalf("parameter %+v, want required path parameter", parameter)
		}
	}

	listWorkspaces, found := findOperation(operations, "list-workspaces")
	if !found {
		t.Fatalf("list-workspaces not found in %v", commands)
	}
	activity := listWorkspaces.Parameters[0]
	if activity.Name != "activity" || activity.In != "query" || activity.Required || !slices.Equal(activity.Enum, []string{"active"}) {
		t.Fatalf("activity parameter = %+v, want optional query enum parameter", activity)
	}
}

func TestKebabCase(t *testing.T) {
	tests := []struct {
		identifier string
		want       string
	}{
		{identifier: "listWorkspaces", want: "list-workspaces"},
		{identifier: "getOpenAPISpec", want: "get-open-api-spec"},
		{identifier: "putWorkspaceTerminal", want: "put-workspace-terminal"},
		{identifier: "workspaceId", want: "workspace-id"},
		{identifier: "scope", want: "scope"},
	}

	for _, test := range tests {
		t.Run(test.identifier, func(t *testing.T) {
			if got := kebabCase(test.identifier); got != test.want {
				t.Fatalf("kebabCase(%q) = %q, want %q", test.identifier, got, test.want)
			}
		})
	}
}

func TestParseOperationsRejectsUnsupportedShapes(t *testing.T) {
	tests := []struct {
		name      string
		spec      string
		wantError string
	}{
		{
			name:      "missing operation ID",
			spec:      `{"paths":{"/api/v1/things":{"get":{}}}}`,
			wantError: "has no operationId",
		},
		{
			name:      "non-string parameter",
			spec:      `{"paths":{"/api/v1/things":{"get":{"operationId":"listThings","parameters":[{"name":"limit","in":"query","type":"integer"}]}}}}`,
			wantError: "unsupported query parameter type integer",
		},
		{
			name:      "unsupported parameter location",
			spec:      `{"paths":{"/api/v1/things":{"post":{"operationId":"createThing","parameters":[{"name":"file","in":"formData","type":"string"}]}}}}`,
			wantError: "unsupported parameter location formData",
		},
		{
			name:      "duplicate body parameters",
			spec:      `{"paths":{"/api/v1/things":{"post":{"operationId":"createThing","parameters":[{"name":"a","in":"body"},{"name":"b","in":"body"}]}}}}`,
			wantError: "more than one body parameter",
		},
		{
			name:      "duplicate command names",
			spec:      `{"paths":{"/api/v1/things":{"get":{"operationId":"listThings"}},"/api/v2/things":{"get":{"operationId":"list-things"}}}}`,
			wantError: "same command list-things",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := parseOperations([]byte(test.spec))
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("parseOperations() error = %v, want error containing %q", err, test.wantError)
			}
		})
	}
}
