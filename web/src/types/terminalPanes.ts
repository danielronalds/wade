export const TerminalPanes = {
  Agent: 'agent',
  Misc: 'misc'
} as const;

export const terminalPanes = [TerminalPanes.Agent, TerminalPanes.Misc] as const;

export type TerminalPaneId = (typeof TerminalPanes)[keyof typeof TerminalPanes];
