import {
  getConnectTerminalSessionUrl,
  type ConnectTerminalSessionParams
} from '@/api/generated/wade';

export const createTerminalWebSocket = (params: ConnectTerminalSessionParams) => {
  const url = new URL(getConnectTerminalSessionUrl(params), window.location.href);
  url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:';

  const socket = new WebSocket(url);
  socket.binaryType = 'arraybuffer';

  return socket;
};
