<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue';
import { TerminalPanes, type TerminalPaneId } from '../types/terminalPanes';
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

const activePane = ref<TerminalPaneId>(TerminalPanes.Agent);
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

watch(combinedConnectionStatus, (status) => {
  emit('connectionStatusChange', status);
}, { immediate: true });
</script>

<template>
  <section id="terminal-tab" aria-label="Terminal screens">
    <TerminalPane
      class="agent-pane"
      :project-name="projectName"
      :terminal-name="TerminalPanes.Agent"
      label="Agent"
      :is-active="isAgentPaneActive"
      @activate="activatePane(TerminalPanes.Agent)"
      @connection-status-change="updateConnectionStatus(TerminalPanes.Agent, $event)"
    />
    <TerminalPane
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
