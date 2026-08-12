// This file is completely vibecoded and needs to be properly reviewed.

package api

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func serverAddress(server *httptest.Server) string {
	return strings.TrimPrefix(server.URL, "http://")
}

func TestRunOperationWritesSuccessResponseToStdout(t *testing.T) {
	responseBody := `{"items":[{"id":"wade"}]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/api/v1/workspaces" {
			t.Errorf("path = %s, want /api/v1/workspaces", r.URL.Path)
		}
		if r.URL.Query().Get("activity") != "active" {
			t.Errorf("activity = %q, want active", r.URL.Query().Get("activity"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, responseBody)
	}))
	defer server.Close()

	var output bytes.Buffer
	controller := newTestController(&output, strings.NewReader(""))

	exitCode, err := controller.HandleArgs([]string{"api", "list-workspaces", "--activity", "active", "--address", serverAddress(server)})
	if err != nil {
		t.Fatalf("HandleArgs() error = %v, want nil", err)
	}
	if exitCode != 0 {
		t.Fatalf("HandleArgs() exit code = %d, want 0", exitCode)
	}
	if output.String() != responseBody {
		t.Fatalf("output = %q, want %q", output.String(), responseBody)
	}
}

func TestRunOperationWritesNothingForNoContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	var output bytes.Buffer
	controller := newTestController(&output, strings.NewReader(""))

	exitCode, err := controller.HandleArgs([]string{
		"api", "delete-workspace-terminal",
		"--workspace-id", "wade",
		"--terminal-id", "agent:pi",
		"--address", serverAddress(server),
	})
	if err != nil {
		t.Fatalf("HandleArgs() error = %v, want nil", err)
	}
	if exitCode != 0 {
		t.Fatalf("HandleArgs() exit code = %d, want 0", exitCode)
	}
	if output.Len() != 0 {
		t.Fatalf("output = %q, want empty", output.String())
	}
}

func TestRunOperationReturnsProblemResponseAsError(t *testing.T) {
	problem := `{"type":"not_found","title":"Workspace not found"}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, problem)
	}))
	defer server.Close()

	var output bytes.Buffer
	controller := newTestController(&output, strings.NewReader(""))

	_, err := controller.HandleArgs([]string{"api", "get-workspace", "--workspace-id", "missing", "--address", serverAddress(server)})
	if err == nil {
		t.Fatal("HandleArgs() error = nil, want problem error")
	}
	for _, expected := range []string{"HTTP 404", "Workspace not found"} {
		if !strings.Contains(err.Error(), expected) {
			t.Fatalf("error %q does not contain %q", err.Error(), expected)
		}
	}
	if output.Len() != 0 {
		t.Fatalf("output = %q, want empty", output.String())
	}
}

func TestRunOperationEscapesPathParameters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != "/api/v1/workspaces/a%2Fb%20c" {
			t.Errorf("escaped path = %s, want /api/v1/workspaces/a%%2Fb%%20c", r.URL.EscapedPath())
		}
		_, _ = io.WriteString(w, `{}`)
	}))
	defer server.Close()

	controller := newTestController(&bytes.Buffer{}, strings.NewReader(""))

	_, err := controller.HandleArgs([]string{"api", "get-workspace", "--workspace-id", "a/b c", "--address", serverAddress(server)})
	if err != nil {
		t.Fatalf("HandleArgs() error = %v, want nil", err)
	}
}

func TestRunOperationSendsBodyFromEachSource(t *testing.T) {
	requestBody := `{"text":"Run the tests\n","mode":"bracketed-paste"}`

	bodyFile := filepath.Join(t.TempDir(), "input.json")
	if err := os.WriteFile(bodyFile, []byte(requestBody), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v, want nil", err)
	}

	tests := []struct {
		name  string
		body  string
		stdin string
	}{
		{name: "inline", body: requestBody},
		{name: "file", body: "@" + bodyFile},
		{name: "stdin", body: "-", stdin: requestBody},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("content type = %q, want application/json", r.Header.Get("Content-Type"))
				}
				received, _ := io.ReadAll(r.Body)
				if string(received) != requestBody {
					t.Errorf("body = %q, want %q", received, requestBody)
				}
				w.WriteHeader(http.StatusNoContent)
			}))
			defer server.Close()

			controller := newTestController(&bytes.Buffer{}, strings.NewReader(test.stdin))

			exitCode, err := controller.HandleArgs([]string{
				"api", "send-workspace-terminal-input",
				"--workspace-id", "wade",
				"--terminal-id", "agent:pi",
				"--body", test.body,
				"--address", serverAddress(server),
			})
			if err != nil {
				t.Fatalf("HandleArgs() error = %v, want nil", err)
			}
			if exitCode != 0 {
				t.Fatalf("HandleArgs() exit code = %d, want 0", exitCode)
			}
		})
	}
}

func TestRunOperationRejectsMissingRequiredFlags(t *testing.T) {
	tests := []struct {
		name      string
		arguments []string
		wantError string
	}{
		{
			name:      "missing path parameter",
			arguments: []string{"api", "get-workspace"},
			wantError: "missing required flag --workspace-id",
		},
		{
			name:      "missing required query parameter",
			arguments: []string{"api", "get-review-snapshot-file-contents", "--snapshot-id", "s1", "--file-id", "f1"},
			wantError: "missing required flag --scope",
		},
		{
			name:      "missing required body",
			arguments: []string{"api", "send-workspace-terminal-input", "--workspace-id", "wade", "--terminal-id", "agent:pi"},
			wantError: "missing required flag --body",
		},
		{
			name:      "unknown flag",
			arguments: []string{"api", "get-settings", "--nope", "1"},
			wantError: "flag provided but not defined",
		},
		{
			name:      "unexpected argument",
			arguments: []string{"api", "get-settings", "extra"},
			wantError: "unexpected argument: extra",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			controller := newTestController(&bytes.Buffer{}, strings.NewReader(""))

			_, err := controller.HandleArgs(test.arguments)
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("HandleArgs() error = %v, want error containing %q", err, test.wantError)
			}
		})
	}
}

func TestRunOperationAddressFlagWinsOverEnvironment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{}`)
	}))
	defer server.Close()

	controller := newTestController(&bytes.Buffer{}, strings.NewReader(""))
	controller.environment = environmentStub{"WADE_ADDR": "unreachable.invalid:1"}

	_, err := controller.HandleArgs([]string{"api", "get-settings", "--address", serverAddress(server)})
	if err != nil {
		t.Fatalf("HandleArgs() error = %v, want nil", err)
	}
}

func TestRunOperationReportsUnreachableServer(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v, want nil", err)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("Close() error = %v, want nil", err)
	}

	controller := newTestController(&bytes.Buffer{}, strings.NewReader(""))

	_, err = controller.HandleArgs([]string{"api", "get-settings", "--address", address})
	if err == nil {
		t.Fatal("HandleArgs() error = nil, want unreachable server error")
	}
	for _, expected := range []string{"cannot reach WADE at http://" + address, "wade server"} {
		if !strings.Contains(err.Error(), expected) {
			t.Fatalf("error %q does not contain %q", err.Error(), expected)
		}
	}
}
