import type { Agent, LinearSettings, Settings } from '@/api/generated/wade';
import { defaultThemeAccentColor, normaliseThemeAccentColor } from '@/utils/theme';

export type { Agent, LinearSettings, Settings };

export const defaultAgents: Agent[] = [
  { name: 'Pi', command: 'pi -c', default: true },
  { name: 'Claude', command: 'claude', default: false }
];

export const createEmptySettings = (): Settings => ({
  workspaceDirectories: [],
  shell: '',
  agents: defaultAgents.map((agent) => ({ ...agent })),
  copyIgnoredFilesOnWorktreeCreation: false,
  openWorktreesInNewTabs: false,
  worktreeCopyExcludes: [],
  themeAccentColor: defaultThemeAccentColor,
  linear: {
    enabled: false,
    workspace: ''
  }
});

export const cloneAgents = (agents: readonly Agent[]): Agent[] => agents.map((agent) => ({ ...agent }));

export const cloneSettings = (settings: Settings): Settings => ({
  workspaceDirectories: [...settings.workspaceDirectories],
  shell: settings.shell,
  agents: cloneAgents(settings.agents),
  copyIgnoredFilesOnWorktreeCreation: settings.copyIgnoredFilesOnWorktreeCreation,
  openWorktreesInNewTabs: settings.openWorktreesInNewTabs,
  worktreeCopyExcludes: [...settings.worktreeCopyExcludes],
  themeAccentColor: settings.themeAccentColor,
  linear: { ...settings.linear }
});

export const normaliseWorkspaceDirectories = (directories: readonly string[]) =>
  directories.map((directory) => directory.trim());

export const normaliseShell = (shell: string) => shell.trim();

export const normaliseAgents = (agents: readonly Agent[]) =>
  agents.map((agent) => ({
    name: agent.name.trim(),
    command: agent.command.trim(),
    default: agent.default
  }));

export const normaliseWorktreeCopyExcludes = (excludes: readonly string[]) =>
  excludes.map((exclude) => exclude.trim()).filter((exclude) => exclude !== '');

export const normaliseSettings = (settings: Settings): Settings => ({
  workspaceDirectories: normaliseWorkspaceDirectories(settings.workspaceDirectories),
  shell: normaliseShell(settings.shell),
  agents: normaliseAgents(settings.agents),
  copyIgnoredFilesOnWorktreeCreation: settings.copyIgnoredFilesOnWorktreeCreation,
  openWorktreesInNewTabs: settings.openWorktreesInNewTabs,
  worktreeCopyExcludes: normaliseWorktreeCopyExcludes(settings.worktreeCopyExcludes),
  themeAccentColor: normaliseThemeAccentColor(settings.themeAccentColor),
  linear: {
    enabled: settings.linear.enabled,
    workspace: settings.linear.workspace.trim()
  }
});

export const isValidLinearWorkspace = (workspace: string) => {
  const trimmedWorkspace = workspace.trim();

  return trimmedWorkspace !== '.' && trimmedWorkspace !== '..' && /^[A-Za-z0-9._~-]+$/.test(trimmedWorkspace);
};

export const isValidShell = (shell: string) => shell.trim().split(/\s+/).filter(Boolean).length <= 1;

export const isValidWorkspaceDirectory = (directory: string) => {
  const trimmedDirectory = directory.trim();

  return trimmedDirectory === '~' || trimmedDirectory.startsWith('~/') || trimmedDirectory.startsWith('/');
};

export const isValidAgents = (agents: readonly Agent[]) => {
  if (agents.length === 0) {
    return false;
  }

  const defaultAgents = agents.filter((agent) => agent.default);
  if (defaultAgents.length !== 1) {
    return false;
  }

  const agentNames = new Set<string>();
  for (const agent of agents) {
    const name = agent.name.trim();
    const command = agent.command.trim();
    if (name === '' || command === '') {
      return false;
    }

    const key = name.toLowerCase();
    if (agentNames.has(key)) {
      return false;
    }

    agentNames.add(key);
  }

  return true;
};
