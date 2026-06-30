export const TerminalPanes = {
  Agent: 'agent',
  Misc: 'misc'
} as const;

export type TerminalPaneId = typeof TerminalPanes[keyof typeof TerminalPanes];
