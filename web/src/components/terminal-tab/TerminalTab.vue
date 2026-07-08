<script setup lang="ts">
import { computed, reactive, ref, toRef, watch } from 'vue';
import { useTerminalTabPaneZoom } from './composable/useTerminalTabPaneZoom';
import { TerminalPanes, type TerminalPaneId } from '../../types/terminalPanes';
import {
  combineTerminalConnectionStatuses,
  createDisconnectedTerminalConnectionStatus,
  type TerminalConnectionStatus
} from '../../types/terminalConnectionStatus';
import CollapsedTerminalRail from './components/CollapsedTerminalRail.vue';
import TerminalPane from '../terminal-pane/TerminalPane.vue';

const props = defineProps<{
  projectName: string;
  isActive: boolean;
}>();

const emit = defineEmits<{
  connectionStatusChange: [status: TerminalConnectionStatus];
}>();

type TerminalPaneComponent = {
  focusTerminal: () => Promise<void>;
};

const agentPane = ref<TerminalPaneComponent | null>(null);
const miscPane = ref<TerminalPaneComponent | null>(null);
const connectionStatuses = reactive<Record<TerminalPaneId, TerminalConnectionStatus>>({
  [TerminalPanes.Agent]: createDisconnectedTerminalConnectionStatus(),
  [TerminalPanes.Misc]: createDisconnectedTerminalConnectionStatus()
});
const combinedConnectionStatus = computed(() => combineTerminalConnectionStatuses([
  connectionStatuses[TerminalPanes.Agent],
  connectionStatuses[TerminalPanes.Misc]
]));

const getPaneComponent = (pane: TerminalPaneId) => pane === TerminalPanes.Misc
  ? miscPane.value
  : agentPane.value;

const focusPane = async (pane: TerminalPaneId) => {
  await getPaneComponent(pane)?.focusTerminal();
};

const {
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
} = useTerminalTabPaneZoom({
  isActive: toRef(props, 'isActive'),
  focusPane
});

const updateConnectionStatus = (pane: TerminalPaneId, status: TerminalConnectionStatus) => {
  connectionStatuses[pane] = status;
};

watch(combinedConnectionStatus, (status) => {
  emit('connectionStatusChange', status);
}, { immediate: true });

defineExpose({
  focusActiveTerminal,
  focusFirstPane,
  switchToNextTerminal,
  toggleActivePaneZoom
});
</script>

<template>
  <section id="terminal-tab" :data-layout="terminalTabLayout" aria-label="Terminal screens">
    <CollapsedTerminalRail
      v-if="isAgentPaneCollapsed"
      label="Agent"
      side="left"
      @restore="restoreSplitView"
    />
    <TerminalPane
      ref="agentPane"
      class="agent-pane"
      :class="{ 'agent-pane--split': terminalTabLayout === 'split' }"
      :project-name="projectName"
      :terminal-name="TerminalPanes.Agent"
      label="Agent"
      :is-active="isAgentPaneActive"
      :is-collapsed="isAgentPaneCollapsed"
      :is-zoomed="isAgentPaneZoomed"
      show-zoom-icon
      @activate="activatePane(TerminalPanes.Agent)"
      @toggle-zoom="togglePaneZoom(TerminalPanes.Agent)"
      @connection-status-change="updateConnectionStatus(TerminalPanes.Agent, $event)"
    />
    <TerminalPane
      ref="miscPane"
      :project-name="projectName"
      :terminal-name="TerminalPanes.Misc"
      label="Misc"
      :is-active="isMiscPaneActive"
      :is-collapsed="isMiscPaneCollapsed"
      :is-zoomed="isMiscPaneZoomed"
      show-zoom-icon
      @activate="activatePane(TerminalPanes.Misc)"
      @toggle-zoom="togglePaneZoom(TerminalPanes.Misc)"
      @connection-status-change="updateConnectionStatus(TerminalPanes.Misc, $event)"
    />
    <CollapsedTerminalRail
      v-if="isMiscPaneCollapsed"
      label="Misc"
      side="right"
      @restore="restoreSplitView"
    />
  </section>
</template>

<style scoped>
#terminal-tab {
  --terminal-header-height: 28px;

  width: 100%;
  height: 100%;
  min-width: 0;
  min-height: 0;
  display: grid;
  grid-template-columns: 80ch minmax(0, 1fr);
  overflow: hidden;
}

#terminal-tab[data-layout="agent-zoomed"] {
  grid-template-columns: minmax(0, 1fr) var(--terminal-header-height);
}

#terminal-tab[data-layout="misc-zoomed"] {
  grid-template-columns: var(--terminal-header-height) minmax(0, 1fr);
}

.agent-pane--split {
  border-right: 1px solid var(--text);
}
</style>
