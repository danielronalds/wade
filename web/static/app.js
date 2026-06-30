import { readRecentProjects, recordRecentProject } from './recent-projects.js';

const homeView = document.getElementById('home-view');
const terminalView = document.getElementById('terminal-view');
const recentProjects = document.getElementById('recent-projects');
const emptyProjects = document.getElementById('empty-projects');
const terminalElement = document.getElementById('terminal');
const connectionStatus = document.getElementById('connection-status');
const connectionText = connectionStatus?.querySelector('span:last-child');
const projectTitle = document.getElementById('project-title');

if (!homeView || !terminalView || !recentProjects || !emptyProjects || !terminalElement || !connectionStatus || !connectionText || !projectTitle) {
  throw new Error('Expected application elements to exist');
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

const getCurrentProject = () => {
  const requestedPath = window.location.pathname.replace(/^\/+|\/+$/g, '');
  if (requestedPath === '') {
    return '';
  }

  return decodeURIComponent(requestedPath);
};

const showHome = () => {
  terminalView.hidden = true;
  homeView.hidden = false;

  const projects = readRecentProjects();
  recentProjects.replaceChildren(
    ...projects.map((project) => {
      const item = document.createElement('li');
      const link = document.createElement('a');
      link.href = `/${encodeURIComponent(project)}`;
      link.textContent = project;
      item.append(link);
      return item;
    })
  );

  emptyProjects.hidden = projects.length > 0;
};

const waitForEmbeddedFont = async () => {
  if (!document.fonts) {
    return;
  }

  await document.fonts.load(`14px "${embeddedFontFamily}"`);
  await document.fonts.ready;
};

const createTerminal = () => new Terminal({
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

const showTerminal = async (projectName) => {
  document.title = `WADE - ${projectName}`;
  projectTitle.textContent = projectName;
  homeView.hidden = true;
  terminalView.hidden = false;

  await waitForEmbeddedFont();

  let socket;
  const terminal = createTerminal();
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
    const params = new URLSearchParams({ project: projectName });
    socket = new WebSocket(`${protocol}//${window.location.host}/ws?${params}`);
    socket.binaryType = 'arraybuffer';

    socket.addEventListener('open', () => {
      recordRecentProject(projectName);
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
};

const projectName = getCurrentProject();
if (projectName === '') {
  showHome();
} else {
  await showTerminal(projectName);
}
