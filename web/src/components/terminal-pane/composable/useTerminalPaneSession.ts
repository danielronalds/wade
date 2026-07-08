import { nextTick, onBeforeUnmount, onMounted, ref, watch, type Ref } from 'vue';
import type { TerminalConnectionStatus } from '../../../types/terminalConnectionStatus';
import { useTerminalSession } from '../../../composables/useTerminalSession';

type TerminalPaneSessionOptions = {
  projectName: string;
  terminalName: string;
  isActive: Readonly<Ref<boolean>>;
  onConnectionStatusChange: (status: TerminalConnectionStatus) => void;
};

export const useTerminalPaneSession = ({
  projectName,
  terminalName,
  isActive,
  onConnectionStatusChange
}: TerminalPaneSessionOptions) => {
  const terminalElement = ref<HTMLElement | null>(null);
  const terminalSession = useTerminalSession({
    projectName,
    terminalName,
    terminalElement,
    isActive
  });

  const publishConnectionStatus = () => {
    onConnectionStatusChange({
      connectionStatusText: terminalSession.connectionStatusText.value,
      isConnected: terminalSession.isConnected.value
    });
  };

  const fitAndFocusTerminal = async () => {
    if (!isActive.value) {
      return;
    }

    await nextTick();

    terminalSession.fitAndResize();
    terminalSession.focusTerminal();
  };

  const reloadTerminal = async () => {
    await terminalSession.reload();
    await fitAndFocusTerminal();
  };

  const scrollTerminalToBottom = async () => {
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
    void terminalSession.start();
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
