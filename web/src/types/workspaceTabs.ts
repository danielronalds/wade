export const WorkspaceTabs = {
  Terminal: 'terminal',
  Server: 'server',
  Review: 'review'
} as const;

export const workspaceTabs = [WorkspaceTabs.Terminal, WorkspaceTabs.Server, WorkspaceTabs.Review] as const;

export type WorkspaceTab = (typeof WorkspaceTabs)[keyof typeof WorkspaceTabs];
