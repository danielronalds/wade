export const ProjectTabs = {
  Terminal: 'terminal',
  Server: 'server',
  Review: 'review'
} as const;

export const projectTabs = [ProjectTabs.Terminal, ProjectTabs.Server, ProjectTabs.Review] as const;

export type ProjectTab = typeof ProjectTabs[keyof typeof ProjectTabs];
