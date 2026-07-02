export type Settings = {
  projectDirectories: string[];
  agentPaneCommand: string;
};

export const createEmptySettings = (): Settings => ({
  projectDirectories: [],
  agentPaneCommand: ''
});

export const cloneSettings = (settings: Settings): Settings => ({
  projectDirectories: [...settings.projectDirectories],
  agentPaneCommand: settings.agentPaneCommand
});

export const normaliseProjectDirectories = (directories: readonly string[]) => directories.map((directory) => directory.trim());

export const normaliseAgentPaneCommand = (command: string) => command.trim();

export const normaliseSettings = (settings: Settings): Settings => ({
  projectDirectories: normaliseProjectDirectories(settings.projectDirectories),
  agentPaneCommand: normaliseAgentPaneCommand(settings.agentPaneCommand)
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
    && typeof response.agentPaneCommand === 'string';
};
