export const ProjectTabs = {
  Terminal: 'terminal',
  Server: 'server'
} as const;

export const projectTabs = [ProjectTabs.Terminal, ProjectTabs.Server] as const;

export type ProjectTab = typeof ProjectTabs[keyof typeof ProjectTabs];
