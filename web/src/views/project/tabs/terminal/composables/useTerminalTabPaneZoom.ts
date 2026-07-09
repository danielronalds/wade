import { computed, nextTick, ref, type Ref } from 'vue';
import { TerminalPanes, terminalPanes, type TerminalPaneId } from '@/types/terminalPanes';

type TerminalTabPaneZoomOptions = {
  isActive: Readonly<Ref<boolean>>;
  focusPane: (pane: TerminalPaneId) => Promise<void>;
};

export const useTerminalTabPaneZoom = ({
  isActive,
  focusPane
}: TerminalTabPaneZoomOptions) => {
  const activePane = ref<TerminalPaneId>(TerminalPanes.Agent);
  const zoomedPane = ref<TerminalPaneId | null>(null);

  const activeVisiblePane = computed(() => zoomedPane.value ?? activePane.value);

  const isAgentPaneZoomed = computed(() => zoomedPane.value === TerminalPanes.Agent);
  const isMiscPaneZoomed = computed(() => zoomedPane.value === TerminalPanes.Misc);

  const isAgentPaneCollapsed = computed(() => isMiscPaneZoomed.value);
  const isMiscPaneCollapsed = computed(() => isAgentPaneZoomed.value);

  const isAgentPaneActive = computed(() => isActive.value
    && !isAgentPaneCollapsed.value
    && activeVisiblePane.value === TerminalPanes.Agent);
  const isMiscPaneActive = computed(() => isActive.value
    && !isMiscPaneCollapsed.value
    && activeVisiblePane.value === TerminalPanes.Misc);

  const terminalTabLayout = computed(() => {
    if (isAgentPaneZoomed.value) {
      return 'agent-zoomed';
    }

    if (isMiscPaneZoomed.value) {
      return 'misc-zoomed';
    }

    return 'split';
  });

  const focusActiveTerminal = async () => {
    if (!isActive.value) {
      return;
    }

    await nextTick();
    await focusPane(activeVisiblePane.value);
  };

  const activatePane = (pane: TerminalPaneId) => {
    if (zoomedPane.value && zoomedPane.value !== pane) {
      return;
    }

    activePane.value = pane;
  };

  const restoreSplitView = async (paneToFocus: TerminalPaneId = zoomedPane.value ?? activePane.value) => {
    zoomedPane.value = null;
    activePane.value = paneToFocus;
    await focusActiveTerminal();
  };

  const zoomPane = async (pane: TerminalPaneId) => {
    activePane.value = pane;
    zoomedPane.value = pane;
    await focusActiveTerminal();
  };

  const togglePaneZoom = async (pane: TerminalPaneId) => {
    if (zoomedPane.value) {
      await restoreSplitView();
      return;
    }

    await zoomPane(pane);
  };

  const toggleActivePaneZoom = async () => {
    if (zoomedPane.value) {
      await restoreSplitView();
      return;
    }

    await zoomPane(activePane.value);
  };

  const focusFirstPane = async () => {
    const firstPane = terminalPanes[0];

    if (zoomedPane.value && zoomedPane.value !== firstPane) {
      zoomedPane.value = null;
    }

    activePane.value = firstPane;
    await focusActiveTerminal();
  };

  const switchToNextTerminal = async () => {
    const activePaneIndex = terminalPanes.indexOf(activeVisiblePane.value);
    const nextPane = terminalPanes[(activePaneIndex + 1) % terminalPanes.length];

    if (zoomedPane.value) {
      await restoreSplitView(nextPane);
      return;
    }

    activePane.value = nextPane;
    await focusActiveTerminal();
  };

  return {
    activatePane,
    focusActiveTerminal,
    focusFirstPane,
    isAgentPaneActive,
    isAgentPaneCollapsed,
    isAgentPaneZoomed,
    isMiscPaneActive,
    isMiscPaneCollapsed,
    isMiscPaneZoomed,
    restoreSplitView,
    switchToNextTerminal,
    terminalTabLayout,
    toggleActivePaneZoom,
    togglePaneZoom
  };
};
