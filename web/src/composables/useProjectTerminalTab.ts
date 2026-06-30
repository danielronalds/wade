import { nextTick, onBeforeUnmount, onMounted, ref, watch, type Ref } from 'vue';
import type { TerminalConnectionStatus } from '../terminalConnectionStatus';
import { useTerminalSession } from './useTerminalSession';

type ProjectTerminalTabOptions = {
  projectName: string;
  terminalName: string;
  isActive: Readonly<Ref<boolean>>;
  onConnectionStatusChange: (status: TerminalConnectionStatus) => void;
};

export const useProjectTerminalTab = ({
  projectName,
  terminalName,
  isActive,
  onConnectionStatusChange
}: ProjectTerminalTabOptions) => {
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
    terminalElement
  };
};
