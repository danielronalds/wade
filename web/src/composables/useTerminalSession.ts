import { readonly, type Ref, ref } from 'vue';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { useRecentProjects } from './useRecentProjects';

type Disposable = {
  dispose: () => void;
};

type TerminalSessionOptions = {
  projectName: string;
  terminalName: string;
  terminalElement: Ref<HTMLElement | null>;
  isActive: Readonly<Ref<boolean>>;
};

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

const quoteFontFamily = (font: string) => font === 'monospace'
  ? font
  : `"${font.replace(/["\\]/g, '')}"`;

const getFontFamily = () => {
  const queryFont = new URLSearchParams(window.location.search).get('font');

  return [queryFont, ...nerdFontStack]
    .filter((font): font is string => Boolean(font))
    .map(quoteFontFamily)
    .join(', ');
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
  fontFamily: getFontFamily(),
  fontSize: 14,
  fontWeight: 400,
  fontWeightBold: 400,
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

export const useTerminalSession = ({
  projectName,
  terminalName,
  terminalElement,
  isActive
}: TerminalSessionOptions) => {
  const { recordRecentProject } = useRecentProjects();
  const isConnected = ref(false);
  const connectionStatusText = ref('Disconnected');

  let socket: WebSocket | undefined;
  let terminal: Terminal | undefined;
  let fitAddon: FitAddon | undefined;
  let resizeObserver: ResizeObserver | undefined;
  let terminalDataDisposable: Disposable | undefined;
  let terminalResizeDisposable: Disposable | undefined;
  let isReloading = false;
  let isStopped = true;
  let reloadingRun: number | undefined;
  let sessionRun = 0;

  const isSessionRunActive = (run: number) => !isStopped && sessionRun === run;

  const setConnectionStatus = (connected: boolean, text: string) => {
    isConnected.value = connected;
    connectionStatusText.value = text;
  };

  const sendResize = () => {
    if (!socket || socket.readyState !== WebSocket.OPEN || !terminal) {
      return;
    }

    socket.send(JSON.stringify({
      type: 'resize',
      cols: terminal.cols,
      rows: terminal.rows
    }));
  };

  const isTerminalVisible = () => Boolean(terminalElement.value?.getClientRects().length);

  const fitAndResize = () => {
    if (!fitAddon || !isTerminalVisible()) {
      return;
    }

    fitAddon.fit();
    sendResize();
  };

  const sendTerminalInput = (data: string) => {
    if (!socket || socket.readyState !== WebSocket.OPEN) {
      return;
    }

    socket.send(encoder.encode(data));
  };

  const focusTerminal = () => {
    if (!isActive.value) {
      return;
    }

    terminal?.focus();
  };

  const sendEscapeKey = () => {
    sendTerminalInput('\x1b');
    focusTerminal();
  };

  const isTerminalKeyboardEvent = (event: KeyboardEvent) => {
    const element = terminalElement.value;

    return Boolean(element && event.composedPath().includes(element));
  };

  const handleTerminalKeyEvent = (event: KeyboardEvent) => {
    if (!isActive.value || event.type !== 'keydown' || event.key !== 'Escape') {
      return true;
    }

    event.preventDefault();
    event.stopPropagation();
    sendEscapeKey();

    return false;
  };

  const handleEscapeKey = (event: KeyboardEvent) => {
    if (!isActive.value || event.key !== 'Escape' || isTerminalKeyboardEvent(event)) {
      return;
    }

    event.preventDefault();
    event.stopImmediatePropagation();
    sendEscapeKey();
  };

  const connectWebSocket = (run: number) => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const params = new URLSearchParams({ project: projectName, terminal: terminalName });
    const connection = new WebSocket(`${protocol}//${window.location.host}/ws?${params}`);
    socket = connection;
    connection.binaryType = 'arraybuffer';

    connection.addEventListener('open', () => {
      if (!isSessionRunActive(run)) {
        return;
      }

      recordRecentProject(projectName);
      setConnectionStatus(true, 'Connected');
      fitAndResize();
      focusTerminal();
    });

    connection.addEventListener('message', (event) => {
      if (!isSessionRunActive(run) || !terminal) {
        return;
      }

      terminal.write(new Uint8Array(event.data));
    });

    connection.addEventListener('close', () => {
      if (!isSessionRunActive(run) || reloadingRun === run) {
        return;
      }

      setConnectionStatus(false, 'Disconnected');
      terminal?.write('\r\nConnection closed.\r\n');
    });

    connection.addEventListener('error', () => {
      if (!isSessionRunActive(run) || reloadingRun === run) {
        return;
      }

      setConnectionStatus(false, 'Error');
      terminal?.write('\r\nConnection error.\r\n');
    });
  };

  const stop = () => {
    isStopped = true;
    sessionRun += 1;
    terminalDataDisposable?.dispose();
    terminalResizeDisposable?.dispose();
    resizeObserver?.disconnect();
    window.removeEventListener('resize', fitAndResize);
    document.removeEventListener('keydown', handleEscapeKey, true);
    socket?.close();
    terminal?.dispose();

    socket = undefined;
    terminal = undefined;
    fitAddon = undefined;
    resizeObserver = undefined;
    terminalDataDisposable = undefined;
    terminalResizeDisposable = undefined;
  };

  const start = async () => {
    stop();
    isStopped = false;
    sessionRun += 1;
    const run = sessionRun;
    setConnectionStatus(false, 'Connecting');
    document.title = `WADE - ${projectName}`;

    await waitForEmbeddedFont();

    if (!isSessionRunActive(run) || !terminalElement.value) {
      return;
    }

    terminal = createTerminal();
    terminal.attachCustomKeyEventHandler(handleTerminalKeyEvent);
    fitAddon = new FitAddon();
    terminal.loadAddon(fitAddon);
    terminal.open(terminalElement.value);

    terminalDataDisposable = terminal.onData(sendTerminalInput);
    terminalResizeDisposable = terminal.onResize(sendResize);
    resizeObserver = new ResizeObserver(fitAndResize);
    resizeObserver.observe(terminalElement.value);
    window.addEventListener('resize', fitAndResize);
    document.addEventListener('keydown', handleEscapeKey, true);

    fitAndResize();
    connectWebSocket(run);
  };

  const closeRemoteTerminal = async () => {
    const params = new URLSearchParams({ project: projectName, terminal: terminalName });
    const response = await fetch(`/api/terminal/reload?${params}`, { method: 'POST' });

    if (!response.ok) {
      throw new Error('Failed to reload terminal');
    }
  };

  const reload = async () => {
    if (isReloading || isStopped) {
      return;
    }

    isReloading = true;
    const run = sessionRun;
    reloadingRun = run;
    setConnectionStatus(false, 'Connecting');

    try {
      await closeRemoteTerminal();

      if (!isSessionRunActive(run)) {
        return;
      }

      await start();
    } catch {
      if (isSessionRunActive(run)) {
        setConnectionStatus(false, 'Error');
        terminal?.write('\r\nReload failed.\r\n');
      }
    } finally {
      if (reloadingRun === run) {
        reloadingRun = undefined;
      }

      isReloading = false;
    }
  };

  return {
    connectionStatusText: readonly(connectionStatusText),
    isConnected: readonly(isConnected),
    fitAndResize,
    focusTerminal,
    reload,
    start,
    stop
  };
};
