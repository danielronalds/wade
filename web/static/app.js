const terminalElement = document.getElementById('terminal');
const connectionStatus = document.getElementById('connection-status');
const connectionText = connectionStatus?.querySelector('span:last-child');

if (!terminalElement || !connectionStatus || !connectionText) {
  throw new Error('Expected terminal elements to exist');
}

const encoder = new TextEncoder();
const embeddedFontFamily = 'WebTerminalJetBrainsMonoNerdFont';
const nerdFontStack = [
  embeddedFontFamily,
  'JetBrainsMono Nerd Font Mono',
  'JetBrainsMono Nerd Font',
  'MesloLGS NF',
  'FiraCode Nerd Font Mono',
  'Hack Nerd Font Mono',
  'Symbols Nerd Font Mono',
  'SFMono-Regular',
  'Menlo',
  'Monaco',
  'Consolas',
  'monospace'
];
const queryFont = new URLSearchParams(window.location.search).get('font');
const fontFamily = [queryFont, ...nerdFontStack]
  .filter(Boolean)
  .map((font) => font === 'monospace' ? font : `"${font.replace(/["\\]/g, '')}"`)
  .join(', ');

let socket;

const waitForEmbeddedFont = async () => {
  if (!document.fonts) {
    return;
  }

  await document.fonts.load(`14px "${embeddedFontFamily}"`);
  await document.fonts.ready;
};

await waitForEmbeddedFont();

const terminal = new Terminal({
  cursorBlink: true,
  cursorStyle: 'block',
  customGlyphs: true,
  fontFamily,
  fontSize: 14,
  letterSpacing: 0,
  lineHeight: 1,
  rescaleOverlappingGlyphs: true,
  scrollback: 10000,
  theme: {
    background: '#17181c',
    foreground: '#f8f8f2',
    cursor: '#f8f8f0',
    cursorAccent: '#17181c',
    selectionBackground: '#45475a',
    black: '#21222c',
    red: '#ff5555',
    green: '#50fa7b',
    yellow: '#f1fa8c',
    blue: '#bd93f9',
    magenta: '#ff79c6',
    cyan: '#8be9fd',
    white: '#f8f8f2',
    brightBlack: '#6272a4',
    brightRed: '#ff6e6e',
    brightGreen: '#69ff94',
    brightYellow: '#ffffa5',
    brightBlue: '#d6acff',
    brightMagenta: '#ff92df',
    brightCyan: '#a4ffff',
    brightWhite: '#ffffff'
  }
});

const fitAddon = new FitAddon.FitAddon();
terminal.loadAddon(fitAddon);
terminal.open(terminalElement);

const updateConnectionStatusLabel = () => {
  const toggleAction = connectionStatus.dataset.open === 'false' ? 'show' : 'hide';
  connectionStatus.setAttribute('aria-label', `${connectionText.textContent}. Click to ${toggleAction} connection status text.`);
};

const setConnectionStatus = (connected, text) => {
  connectionStatus.dataset.connected = String(connected);
  connectionText.textContent = text;
  updateConnectionStatusLabel();
};

const setConnectionStatusOpen = (open) => {
  connectionStatus.dataset.open = String(open);
  connectionStatus.setAttribute('aria-expanded', String(open));
  updateConnectionStatusLabel();
  terminal.focus();
};

const toggleConnectionStatusOpen = () => {
  setConnectionStatusOpen(connectionStatus.dataset.open === 'false');
};

const sendResize = () => {
  if (!socket || socket.readyState !== WebSocket.OPEN) {
    return;
  }

  socket.send(JSON.stringify({
    type: 'resize',
    cols: terminal.cols,
    rows: terminal.rows
  }));
};

const fitAndResize = () => {
  fitAddon.fit();
  sendResize();
};

const sendTerminalInput = (data) => {
  if (!socket || socket.readyState !== WebSocket.OPEN) {
    return;
  }

  socket.send(encoder.encode(data));
};

const handleEscapeKey = (event) => {
  if (event.key !== 'Escape') {
    return;
  }

  event.preventDefault();
  event.stopPropagation();
  sendTerminalInput('\x1b');
  terminal.focus();
};

const connectWebSocket = () => {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  socket = new WebSocket(`${protocol}//${window.location.host}/ws`);
  socket.binaryType = 'arraybuffer';

  socket.addEventListener('open', () => {
    setConnectionStatus(true, 'Connected');
    fitAndResize();
    terminal.focus();
  });

  socket.addEventListener('message', (event) => {
    terminal.write(new Uint8Array(event.data));
  });

  socket.addEventListener('close', () => {
    setConnectionStatus(false, 'Disconnected');
    terminal.write('\r\nConnection closed.\r\n');
  });

  socket.addEventListener('error', () => {
    setConnectionStatus(false, 'Error');
    terminal.write('\r\nConnection error.\r\n');
  });
};

terminal.onData(sendTerminalInput);
connectionStatus.addEventListener('click', toggleConnectionStatusOpen);
document.addEventListener('keydown', handleEscapeKey, true);

terminal.onResize(sendResize);
new ResizeObserver(fitAndResize).observe(terminalElement);
window.addEventListener('resize', fitAndResize);

fitAndResize();
setConnectionStatus(false, 'Connecting');
connectWebSocket();
