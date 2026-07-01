<script setup lang="ts">
import { computed } from 'vue';
import { useProjectTerminalTab } from '../composables/useProjectTerminalTab';
import { ProjectTabs } from '../types/projectTabs';
import TerminalHeader from './TerminalHeader.vue';
import type { TerminalConnectionStatus } from '../types/terminalConnectionStatus';

const props = defineProps<{
  projectName: string;
  isActive: boolean;
}>();

const emit = defineEmits<{
  connectionStatusChange: [status: TerminalConnectionStatus];
}>();

const serverLabel = 'Server';
const isActive = computed(() => props.isActive);
const {
  focusTerminal,
  reloadTerminal,
  scrollTerminalToBottom,
  terminalElement
} = useProjectTerminalTab({
  projectName: props.projectName,
  terminalName: ProjectTabs.Server,
  isActive,
  onConnectionStatusChange: (status) => emit('connectionStatusChange', status)
});

const scrollToBottom = () => {
  void scrollTerminalToBottom();
};

const reload = () => {
  void reloadTerminal();
};

const focusActiveTerminal = async () => {
  await focusTerminal();
};

const switchToNextTerminal = async () => {
  await focusTerminal();
};

defineExpose({
  focusActiveTerminal,
  switchToNextTerminal
});
</script>

<template>
  <section class="server-tab" aria-label="Server terminal pane">
    <TerminalHeader
      :label="serverLabel"
      :is-active="isActive"
      @scroll-to-bottom="scrollToBottom"
      @reload="reload"
    />
    <section ref="terminalElement" class="terminal-screen" aria-label="Server shell"></section>
  </section>
</template>

<style scoped>
.server-tab {
  width: 100%;
  height: 100%;
  min-width: 0;
  min-height: 0;
  display: grid;
  grid-template-rows: 28px minmax(0, 1fr);
  overflow: hidden;
  background: var(--window);
}
</style>
