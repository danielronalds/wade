import { readonly, type Ref, ref } from 'vue';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { WebLinksAddon } from '@xterm/addon-web-links';
import { WebglAddon } from '@xterm/addon-webgl';
import { reloadTerminalSession } from '@/api/generated/wade';
import { useRecentProjects } from '@/features/projects/composables/useRecentProjects';

type Disposable = {
  dispose: () => void;
};

type TerminalSessionOptions = {
  projectName: string;
  terminalName: string;
  agentName?: string;
  terminalElement: Ref<HTMLElement | null>;
  isActive: Readonly<Ref<boolean>>;
  onSessionEnd?: () => void;
};

type TerminalControlMessage = {
  type?: string;
};

const encoder = new TextEncoder();
const replayStartMessageType = 'replayStart';
const replayEndMessageType = 'replayEnd';
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

const openHttpLink = (_event: MouseEvent, uri: string) => {
  if (!URL.canParse(uri)) {
    return;
  }

  const url = new URL(uri);

  if (url.protocol !== 'http:' && url.protocol !== 'https:') {
    return;
  }

  window.open(url.toString(), '_blank', 'noopener,noreferrer');
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
  agentName,
  terminalElement,
  isActive,
  onSessionEnd
}: TerminalSessionOptions) => {
  const { recordRecentProject } = useRecentProjects();
  const isConnected = ref(false);
  const connectionStatusText = ref('Disconnected');

  let socket: WebSocket | undefined;
  let terminal: Terminal | undefined;
  let fitAddon: FitAddon | undefined;
  let webglAddon: WebglAddon | undefined;
  let webglContextLossDisposable: Disposable | undefined;
  let resizeObserver: ResizeObserver | undefined;
  let terminalDataDisposable: Disposable | undefined;
  let terminalResizeDisposable: Disposable | undefined;
  let isReloading = false;
  let isStopped = true;
  let reloadingRun: number | undefined;
  let sessionRun = 0;
  let isReceivingReplay = false;
  let pendingReplayWrites = 0;
  let queuedLiveOutput: Uint8Array[] = [];

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

  const disposeWebglAddon = () => {
    webglContextLossDisposable?.dispose();
    webglAddon?.dispose();
    webglContextLossDisposable = undefined;
    webglAddon = undefined;
  };

  const loadWebglAddon = (activeTerminal: Terminal) => {
    let addon: WebglAddon | undefined;
    let contextLossDisposable: Disposable | undefined;

    try {
      addon = new WebglAddon();
      contextLossDisposable = addon.onContextLoss(disposeWebglAddon);
      activeTerminal.loadAddon(addon);
    } catch {
      contextLossDisposable?.dispose();
      addon?.dispose();
      return;
    }

    webglAddon = addon;
    webglContextLossDisposable = contextLossDisposable;
  };

  const hasReplayInProgress = () => isReceivingReplay || pendingReplayWrites > 0;

  const resetReplayState = () => {
    isReceivingReplay = false;
    pendingReplayWrites = 0;
    queuedLiveOutput = [];
  };

  const flushQueuedLiveOutput = (run: number) => {
    if (!isSessionRunActive(run) || hasReplayInProgress() || !terminal) {
      return;
    }

    const outputs = queuedLiveOutput;
    queuedLiveOutput = [];

    for (const output of outputs) {
      terminal.write(output);
    }
  };

  const writeReplayOutput = (run: number, output: Uint8Array) => {
    if (!terminal) {
      return;
    }

    pendingReplayWrites += 1;
    terminal.write(output, () => {
      if (!isSessionRunActive(run)) {
        return;
      }

      pendingReplayWrites = Math.max(0, pendingReplayWrites - 1);
      flushQueuedLiveOutput(run);
    });
  };

  const writeTerminalOutput = (run: number, output: Uint8Array) => {
    if (isReceivingReplay) {
      writeReplayOutput(run, output);
      return;
    }

    if (hasReplayInProgress()) {
      queuedLiveOutput.push(output);
      return;
    }

    terminal?.write(output);
  };

  const handleTerminalControlMessage = (run: number, data: string) => {
    let parsedMessage: unknown;

    try {
      parsedMessage = JSON.parse(data);
    } catch {
      return;
    }

    if (!parsedMessage || typeof parsedMessage !== 'object' || !('type' in parsedMessage)) {
      return;
    }

    const message = parsedMessage as TerminalControlMessage;

    if (message.type === replayStartMessageType) {
      isReceivingReplay = true;
      return;
    }

    if (message.type === replayEndMessageType) {
      isReceivingReplay = false;
      flushQueuedLiveOutput(run);
    }
  };

  const sendTerminalInput = (data: string) => {
    if (hasReplayInProgress() || !socket || socket.readyState !== WebSocket.OPEN) {
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

  const scrollToBottom = () => {
    terminal?.scrollToBottom();
    focusTerminal();
  };

  const sendEscapeKey = () => {
    sendTerminalInput('\x1b');
    focusTerminal();
  };

  const handleEscapeKey = (event: KeyboardEvent) => {
    if (!isActive.value || event.key !== 'Escape') {
      return;
    }

    event.preventDefault();
    event.stopImmediatePropagation();
    sendEscapeKey();
  };

  const terminalRequestParams = () => {
    const params = new URLSearchParams({ project: projectName, terminal: terminalName });
    if (agentName) {
      params.set('agent', agentName);
    }

    return params;
  };

  const connectWebSocket = (run: number) => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const params = terminalRequestParams();
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

      if (typeof event.data === 'string') {
        handleTerminalControlMessage(run, event.data);
        return;
      }

      writeTerminalOutput(run, new Uint8Array(event.data));
    });

    connection.addEventListener('close', () => {
      if (!isSessionRunActive(run) || reloadingRun === run) {
        return;
      }

      setConnectionStatus(false, 'Disconnected');
      terminal?.write('\r\nConnection closed.\r\n');
      onSessionEnd?.();
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
    resetReplayState();
    terminalResizeDisposable?.dispose();
    resizeObserver?.disconnect();
    disposeWebglAddon();
    window.removeEventListener('resize', fitAndResize);
    document.removeEventListener('keydown', handleEscapeKey, true);
    socket?.close();
    terminal?.dispose();

    socket = undefined;
    terminal = undefined;
    fitAddon = undefined;
    webglAddon = undefined;
    webglContextLossDisposable = undefined;
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
    fitAddon = new FitAddon();
    terminal.loadAddon(fitAddon);
    terminal.loadAddon(new WebLinksAddon(openHttpLink));
    terminal.open(terminalElement.value);
    loadWebglAddon(terminal);

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
    await reloadTerminalSession({ project: projectName, terminal: terminalName, agent: agentName });
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
    } catch (error) {
      if (isSessionRunActive(run)) {
        const message = error instanceof Error ? `: ${error.message}` : '';

        setConnectionStatus(false, 'Error');
        terminal?.write(`\r\nReload failed${message}.\r\n`);
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
    scrollToBottom,
    start,
    stop
  };
};
