import { onBeforeUnmount, onMounted } from 'vue';

const workspaceShortcutPrefixTimeoutMs = 1500;
const workspaceShortcutPrefixKey = 'b';
const nextTerminalPaneKey = 'o';
const terminalZoomKey = 'z';
const scratchpadTerminalKey = 't';
const scratchpadTerminalCode = 'KeyT';

type WorkspaceKeyboardShortcutsOptions = {
  selectSidebarItemBySlot: (slot: number) => void;
  switchToNextTerminal: () => void;
  toggleScratchpadTerminal: () => void;
  toggleTerminalZoom: () => void;
};

const isCtrlShortcut = (event: KeyboardEvent, key: string) => event.ctrlKey
  && !event.altKey
  && !event.metaKey
  && !event.shiftKey
  && event.key.toLowerCase() === key;

const isCtrlAltShortcut = (event: KeyboardEvent, key: string, code?: string) => event.ctrlKey
  && event.altKey
  && !event.metaKey
  && !event.shiftKey
  && (event.key.toLowerCase() === key || event.code === code);

const isShortcutPrefix = (event: KeyboardEvent) => isCtrlShortcut(event, workspaceShortcutPrefixKey);

const isNumberKey = (key: string) => /^[1-9]$/.test(key);

export const useWorkspaceKeyboardShortcuts = ({
  selectSidebarItemBySlot,
  switchToNextTerminal,
  toggleScratchpadTerminal,
  toggleTerminalZoom
}: WorkspaceKeyboardShortcutsOptions) => {
  let isWaitingForPrefixCommand = false;
  let prefixTimeout: number | undefined;

  const stopEvent = (event: KeyboardEvent) => {
    event.preventDefault();
    event.stopImmediatePropagation();
  };

  const clearPrefix = () => {
    isWaitingForPrefixCommand = false;

    if (prefixTimeout === undefined) {
      return;
    }

    window.clearTimeout(prefixTimeout);
    prefixTimeout = undefined;
  };

  const startPrefix = () => {
    clearPrefix();
    isWaitingForPrefixCommand = true;
    prefixTimeout = window.setTimeout(clearPrefix, workspaceShortcutPrefixTimeoutMs);
  };

  const runPrefixCommand = (event: KeyboardEvent) => {
    const key = event.key.toLowerCase();
    clearPrefix();

    if (key === nextTerminalPaneKey) {
      switchToNextTerminal();
      return;
    }

    if (key === terminalZoomKey) {
      toggleTerminalZoom();
      return;
    }

    if (isNumberKey(key)) {
      selectSidebarItemBySlot(Number(key));
    }
  };

  const handleKeydown = (event: KeyboardEvent) => {
    if (isCtrlAltShortcut(event, scratchpadTerminalKey, scratchpadTerminalCode)) {
      stopEvent(event);
      clearPrefix();
      toggleScratchpadTerminal();
      return;
    }

    if (isShortcutPrefix(event)) {
      stopEvent(event);
      startPrefix();
      return;
    }

    if (!isWaitingForPrefixCommand) {
      return;
    }

    stopEvent(event);
    runPrefixCommand(event);
  };

  onMounted(() => {
    window.addEventListener('keydown', handleKeydown, true);
  });

  onBeforeUnmount(() => {
    clearPrefix();
    window.removeEventListener('keydown', handleKeydown, true);
  });
};
