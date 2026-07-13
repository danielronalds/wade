import { nextTick, onBeforeUnmount, onMounted, ref, watch, type Ref } from 'vue';
import type { TerminalConnectionStatus } from '@/types/terminalConnectionStatus';
import { useTerminalSession } from '@/features/terminal-session/composables/useTerminalSession';

type TerminalPaneSessionOptions = {
  projectName: string;
  terminalName: string;
  agentName?: string;
  isActive: Readonly<Ref<boolean>>;
  isSelectedAgent: Readonly<Ref<boolean>>;
  lazy?: boolean;
  onConnectionStatusChange: (status: TerminalConnectionStatus) => void;
  onSessionEnd?: () => void;
};

export const useTerminalPaneSession = ({
  projectName,
  terminalName,
  agentName,
  isActive,
  isSelectedAgent,
  lazy = false,
  onConnectionStatusChange,
  onSessionEnd
}: TerminalPaneSessionOptions) => {
  const terminalElement = ref<HTMLElement | null>(null);
  const terminalSession = useTerminalSession({
    projectName,
    terminalName,
    agentName,
    terminalElement,
    isActive,
    isSelectedAgent,
    onSessionEnd
  });

  let hasStarted = false;
  let startPromise: Promise<void> | undefined;

  const publishConnectionStatus = () => {
    onConnectionStatusChange({
      connectionStatusText: terminalSession.connectionStatusText.value,
      isConnected: terminalSession.isConnected.value
    });
  };

  const startTerminal = async () => {
    if (hasStarted) {
      return;
    }

    if (startPromise) {
      await startPromise;
      return;
    }

    startPromise = terminalSession.start().finally(() => {
      startPromise = undefined;
    });
    hasStarted = true;
    await startPromise;
  };

  const fitAndFocusTerminal = async () => {
    if (!isActive.value) {
      return;
    }

    if (lazy) {
      await startTerminal();
    }

    await nextTick();

    terminalSession.fitAndResize();
    terminalSession.focusTerminal();
  };

  const reloadTerminal = async () => {
    if (lazy) {
      await startTerminal();
    }

    await terminalSession.reload();
    await fitAndFocusTerminal();
  };

  const scrollTerminalToBottom = async () => {
    if (lazy) {
      await startTerminal();
    }

    await nextTick();
    terminalSession.scrollToBottom();
  };

  watch([terminalSession.connectionStatusText, terminalSession.isConnected], () => {
    publishConnectionStatus();
  }, { immediate: true });

  watch(isActive, (active) => {
    if (!active) {
      return;
    }

    void fitAndFocusTerminal();
  });

  onMounted(() => {
    if (!lazy) {
      hasStarted = true;
      void terminalSession.start();
    }

    void fitAndFocusTerminal();
  });

  onBeforeUnmount(() => {
    terminalSession.stop();
  });

  return {
    focusTerminal: fitAndFocusTerminal,
    reloadTerminal,
    scrollTerminalToBottom,
    terminalElement
  };
};
