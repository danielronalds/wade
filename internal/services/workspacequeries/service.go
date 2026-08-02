package workspacequeries

import (
	"context"

	"wade/internal/services/workspaces"
)

type WorkspaceService interface {
	List(ctx context.Context) ([]workspaces.WorkspaceSummary, error)
	ListByIDs(ctx context.Context, workspaceIDs []string) ([]workspaces.WorkspaceSummary, error)
	Get(ctx context.Context, workspaceID string) (workspaces.Workspace, error)
}

type TerminalActivity interface {
	ActiveTerminalCount(workspaceID string) int
	ActiveWorkspaceIDs() []string
}

type Service struct {
	workspaces WorkspaceService
	activity   TerminalActivity
}

func NewService(workspaceService WorkspaceService, terminalActivity TerminalActivity) Service {
	return Service{workspaces: workspaceService, activity: terminalActivity}
}

func (s Service) List(ctx context.Context) ([]workspaces.WorkspaceSummary, error) {
	workspaceSummaries, err := s.workspaces.List(ctx)
	if err != nil {
		return nil, err
	}

	for index := range workspaceSummaries {
		workspaceSummaries[index].Activity.ActiveTerminalCount = s.activity.ActiveTerminalCount(workspaceSummaries[index].ID)
	}

	return workspaceSummaries, nil
}

func (s Service) ListActive(ctx context.Context) ([]workspaces.WorkspaceSummary, error) {
	activeWorkspaceIDs := s.activity.ActiveWorkspaceIDs()
	if len(activeWorkspaceIDs) == 0 {
		return []workspaces.WorkspaceSummary{}, nil
	}

	workspaceSummaries, err := s.workspaces.ListByIDs(ctx, activeWorkspaceIDs)
	if err != nil {
		return nil, err
	}

	activeWorkspaceSummaries := make([]workspaces.WorkspaceSummary, 0, len(workspaceSummaries))
	for _, workspaceSummary := range workspaceSummaries {
		activeTerminalCount := s.activity.ActiveTerminalCount(workspaceSummary.ID)
		if activeTerminalCount == 0 {
			continue
		}

		workspaceSummary.Activity.ActiveTerminalCount = activeTerminalCount
		activeWorkspaceSummaries = append(activeWorkspaceSummaries, workspaceSummary)
	}

	return activeWorkspaceSummaries, nil
}

func (s Service) Get(ctx context.Context, workspaceID string) (workspaces.Workspace, error) {
	workspace, err := s.workspaces.Get(ctx, workspaceID)
	if err != nil {
		return workspaces.Workspace{}, err
	}

	workspace.Activity.ActiveTerminalCount = s.activity.ActiveTerminalCount(workspaceID)
	return workspace, nil
}
