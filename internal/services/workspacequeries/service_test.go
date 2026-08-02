package workspacequeries

import (
	"context"
	"reflect"
	"testing"

	"wade/internal/services/workspaces"
)

type workspaceServiceStub struct {
	workspaceSummaries []workspaces.WorkspaceSummary
	workspace          workspaces.Workspace
	requestedIDs       *[]string
}

func (s workspaceServiceStub) List(context.Context) ([]workspaces.WorkspaceSummary, error) {
	return append([]workspaces.WorkspaceSummary(nil), s.workspaceSummaries...), nil
}

func (s workspaceServiceStub) ListByIDs(_ context.Context, workspaceIDs []string) ([]workspaces.WorkspaceSummary, error) {
	if s.requestedIDs != nil {
		*s.requestedIDs = append([]string(nil), workspaceIDs...)
	}
	return append([]workspaces.WorkspaceSummary(nil), s.workspaceSummaries...), nil
}

func (s workspaceServiceStub) Get(context.Context, string) (workspaces.Workspace, error) {
	return s.workspace, nil
}

type terminalActivityStub struct {
	activeWorkspaceIDs  []string
	activeTerminalCount map[string]int
}

func (s terminalActivityStub) ActiveTerminalCount(workspaceID string) int {
	return s.activeTerminalCount[workspaceID]
}

func (s terminalActivityStub) ActiveWorkspaceIDs() []string {
	return append([]string(nil), s.activeWorkspaceIDs...)
}

func TestServiceListAddsTerminalActivity(t *testing.T) {
	service := NewService(
		workspaceServiceStub{workspaceSummaries: []workspaces.WorkspaceSummary{
			{ID: "alpha", Name: "alpha"},
			{ID: "bravo", Name: "bravo"},
		}},
		terminalActivityStub{activeTerminalCount: map[string]int{"alpha": 2}},
	)

	got, err := service.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v, want nil", err)
	}

	want := []workspaces.WorkspaceSummary{
		{ID: "alpha", Name: "alpha", Activity: workspaces.WorkspaceActivity{ActiveTerminalCount: 2}},
		{ID: "bravo", Name: "bravo"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("List() = %#v, want %#v", got, want)
	}
}

func TestServiceListActiveRequestsOnlyActiveWorkspaces(t *testing.T) {
	requestedIDs := make([]string, 0)
	service := NewService(
		workspaceServiceStub{
			workspaceSummaries: []workspaces.WorkspaceSummary{
				{ID: "alpha", Name: "alpha"},
				{ID: "bravo", Name: "bravo"},
			},
			requestedIDs: &requestedIDs,
		},
		terminalActivityStub{
			activeWorkspaceIDs:  []string{"alpha", "bravo"},
			activeTerminalCount: map[string]int{"alpha": 2},
		},
	)

	got, err := service.ListActive(context.Background())
	if err != nil {
		t.Fatalf("ListActive() error = %v, want nil", err)
	}

	want := []workspaces.WorkspaceSummary{{
		ID:       "alpha",
		Name:     "alpha",
		Activity: workspaces.WorkspaceActivity{ActiveTerminalCount: 2},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ListActive() = %#v, want %#v", got, want)
	}
	if !reflect.DeepEqual(requestedIDs, []string{"alpha", "bravo"}) {
		t.Fatalf("requested workspace IDs = %#v, want alpha and bravo", requestedIDs)
	}
}

func TestServiceGetAddsTerminalActivity(t *testing.T) {
	service := NewService(
		workspaceServiceStub{workspace: workspaces.Workspace{ID: "alpha", Name: "alpha"}},
		terminalActivityStub{activeTerminalCount: map[string]int{"alpha": 3}},
	)

	got, err := service.Get(context.Background(), "alpha")
	if err != nil {
		t.Fatalf("Get() error = %v, want nil", err)
	}
	if got.Activity.ActiveTerminalCount != 3 {
		t.Fatalf("active terminal count = %d, want 3", got.Activity.ActiveTerminalCount)
	}
}
