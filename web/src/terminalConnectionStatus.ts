export type TerminalConnectionStatus = {
  connectionStatusText: string;
  isConnected: boolean;
};

export const createDisconnectedTerminalConnectionStatus = (): TerminalConnectionStatus => ({
  connectionStatusText: 'Disconnected',
  isConnected: false
});
