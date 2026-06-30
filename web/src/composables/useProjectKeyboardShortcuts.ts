import { onBeforeUnmount, onMounted } from 'vue';

const projectShortcutPrefixTimeoutMs = 1500;

type ProjectKeyboardShortcutsOptions = {
  selectTabBySlot: (slot: number) => void;
  switchToNextTerminal: () => void;
};

const isShortcutPrefix = (event: KeyboardEvent) => event.ctrlKey
  && !event.altKey
  && !event.metaKey
  && event.key.toLowerCase() === 'b';

const isNumberKey = (key: string) => /^[1-9]$/.test(key);

export const useProjectKeyboardShortcuts = ({
  selectTabBySlot,
  switchToNextTerminal
}: ProjectKeyboardShortcutsOptions) => {
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
    prefixTimeout = window.setTimeout(clearPrefix, projectShortcutPrefixTimeoutMs);
  };

  const runPrefixCommand = (event: KeyboardEvent) => {
    const key = event.key.toLowerCase();
    clearPrefix();

    if (key === 'o') {
      switchToNextTerminal();
      return;
    }

    if (isNumberKey(key)) {
      selectTabBySlot(Number(key));
    }
  };

  const handleKeydown = (event: KeyboardEvent) => {
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
