export const createTerminalWebSocket = (socketUrl: string) => {
  const url = new URL(socketUrl, window.location.href);
  url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:';

  const socket = new WebSocket(url);
  socket.binaryType = 'arraybuffer';

  return socket;
};
