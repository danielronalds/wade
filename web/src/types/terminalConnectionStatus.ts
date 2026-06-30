export type TerminalConnectionStatus = {
  connectionStatusText: string;
  isConnected: boolean;
};

export const createDisconnectedTerminalConnectionStatus = (): TerminalConnectionStatus => ({
  connectionStatusText: 'Disconnected',
  isConnected: false
});

export const combineTerminalConnectionStatuses = (statuses: TerminalConnectionStatus[]): TerminalConnectionStatus => {
  if (statuses.length === 0) {
    return createDisconnectedTerminalConnectionStatus();
  }

  if (statuses.some((status) => status.connectionStatusText === 'Error')) {
    return {
      connectionStatusText: 'Error',
      isConnected: false
    };
  }

  if (statuses.some((status) => status.connectionStatusText === 'Connecting')) {
    return {
      connectionStatusText: 'Connecting',
      isConnected: false
    };
  }

  if (statuses.every((status) => status.isConnected)) {
    return {
      connectionStatusText: 'Connected',
      isConnected: true
    };
  }

  return createDisconnectedTerminalConnectionStatus();
};
