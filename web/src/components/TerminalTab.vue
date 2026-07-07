<script setup lang="ts">
import { computed, nextTick, reactive, ref, watch } from 'vue';
import { TerminalPanes, terminalPanes, type TerminalPaneId } from '../types/terminalPanes';
import {
  combineTerminalConnectionStatuses,
  createDisconnectedTerminalConnectionStatus,
  type TerminalConnectionStatus
} from '../types/terminalConnectionStatus';
import TerminalPane from './TerminalPane.vue';

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

const activePane = ref<TerminalPaneId>(TerminalPanes.Agent);
const agentPane = ref<TerminalPaneComponent | null>(null);
const miscPane = ref<TerminalPaneComponent | null>(null);
const connectionStatuses = reactive<Record<TerminalPaneId, TerminalConnectionStatus>>({
  [TerminalPanes.Agent]: createDisconnectedTerminalConnectionStatus(),
  [TerminalPanes.Misc]: createDisconnectedTerminalConnectionStatus()
});
const isAgentPaneActive = computed(() => props.isActive && activePane.value === TerminalPanes.Agent);
const isMiscPaneActive = computed(() => props.isActive && activePane.value === TerminalPanes.Misc);
const combinedConnectionStatus = computed(() => combineTerminalConnectionStatuses([
  connectionStatuses[TerminalPanes.Agent],
  connectionStatuses[TerminalPanes.Misc]
]));

const activatePane = (pane: TerminalPaneId) => {
  activePane.value = pane;
};

const updateConnectionStatus = (pane: TerminalPaneId, status: TerminalConnectionStatus) => {
  connectionStatuses[pane] = status;
};

const getActivePaneComponent = () => activePane.value === TerminalPanes.Misc
  ? miscPane.value
  : agentPane.value;

const focusActiveTerminal = async () => {
  if (!props.isActive) {
    return;
  }

  await nextTick();
  await getActivePaneComponent()?.focusTerminal();
};

const focusFirstPane = async () => {
  activePane.value = terminalPanes[0];
  await focusActiveTerminal();
};

const switchToNextTerminal = async () => {
  const activePaneIndex = terminalPanes.indexOf(activePane.value);
  activePane.value = terminalPanes[(activePaneIndex + 1) % terminalPanes.length];
  await focusActiveTerminal();
};

watch(combinedConnectionStatus, (status) => {
  emit('connectionStatusChange', status);
}, { immediate: true });

defineExpose({
  focusActiveTerminal,
  focusFirstPane,
  switchToNextTerminal
});
</script>

<template>
  <section id="terminal-tab" aria-label="Terminal screens">
    <TerminalPane
      ref="agentPane"
      class="agent-pane"
      :project-name="projectName"
      :terminal-name="TerminalPanes.Agent"
      label="Agent"
      :is-active="isAgentPaneActive"
      @activate="activatePane(TerminalPanes.Agent)"
      @connection-status-change="updateConnectionStatus(TerminalPanes.Agent, $event)"
    />
    <TerminalPane
      ref="miscPane"
      :project-name="projectName"
      :terminal-name="TerminalPanes.Misc"
      label="Misc"
      :is-active="isMiscPaneActive"
      @activate="activatePane(TerminalPanes.Misc)"
      @connection-status-change="updateConnectionStatus(TerminalPanes.Misc, $event)"
    />
  </section>
</template>

<style scoped>
#terminal-tab {
  width: 100%;
  height: 100%;
  min-width: 0;
  min-height: 0;
  display: grid;
  grid-template-columns: 80ch minmax(0, 1fr);
  overflow: hidden;
}

.agent-pane {
  border-right: 1px solid var(--text);
}
</style>
