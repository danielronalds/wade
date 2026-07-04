import { defaultThemeAccentColor, isThemeAccentColor, normaliseThemeAccentColor, type ThemeAccentColor } from '../theme';

export type Settings = {
  projectDirectories: string[];
  agentPaneCommand: string;
  copyIgnoredFilesOnWorktreeCreation: boolean;
  worktreeCopyExcludes: string[];
  themeAccentColor: ThemeAccentColor;
};

export const createEmptySettings = (): Settings => ({
  projectDirectories: [],
  agentPaneCommand: '',
  copyIgnoredFilesOnWorktreeCreation: false,
  worktreeCopyExcludes: [],
  themeAccentColor: defaultThemeAccentColor
});

export const cloneSettings = (settings: Settings): Settings => ({
  projectDirectories: [...settings.projectDirectories],
  agentPaneCommand: settings.agentPaneCommand,
  copyIgnoredFilesOnWorktreeCreation: settings.copyIgnoredFilesOnWorktreeCreation,
  worktreeCopyExcludes: [...settings.worktreeCopyExcludes],
  themeAccentColor: settings.themeAccentColor
});

export const normaliseProjectDirectories = (directories: readonly string[]) => directories.map((directory) => directory.trim());

export const normaliseAgentPaneCommand = (command: string) => command.trim();

export const normaliseWorktreeCopyExcludes = (excludes: readonly string[]) => excludes
  .map((exclude) => exclude.trim())
  .filter((exclude) => exclude !== '');

export const normaliseSettings = (settings: Settings): Settings => ({
  projectDirectories: normaliseProjectDirectories(settings.projectDirectories),
  agentPaneCommand: normaliseAgentPaneCommand(settings.agentPaneCommand),
  copyIgnoredFilesOnWorktreeCreation: settings.copyIgnoredFilesOnWorktreeCreation,
  worktreeCopyExcludes: normaliseWorktreeCopyExcludes(settings.worktreeCopyExcludes),
  themeAccentColor: normaliseThemeAccentColor(settings.themeAccentColor)
});

export const isValidProjectDirectory = (directory: string) => {
  const trimmedDirectory = directory.trim();

  return trimmedDirectory === '~'
    || trimmedDirectory.startsWith('~/')
    || trimmedDirectory.startsWith('/');
};

export const isSettings = (value: unknown): value is Settings => {
  if (!value || typeof value !== 'object') {
    return false;
  }

  const response = value as Partial<Settings>;

  return Array.isArray(response.projectDirectories)
    && response.projectDirectories.every((directory) => typeof directory === 'string')
    && typeof response.agentPaneCommand === 'string'
    && typeof response.copyIgnoredFilesOnWorktreeCreation === 'boolean'
    && Array.isArray(response.worktreeCopyExcludes)
    && response.worktreeCopyExcludes.every((exclude) => typeof exclude === 'string')
    && isThemeAccentColor(response.themeAccentColor);
};
