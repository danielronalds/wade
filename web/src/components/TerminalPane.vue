<script setup lang="ts">
import { computed } from 'vue';
import { useProjectTerminalTab } from '../composables/useProjectTerminalTab';
import TerminalHeader from './TerminalHeader.vue';
import type { TerminalConnectionStatus } from '../types/terminalConnectionStatus';

const props = withDefaults(defineProps<{
  projectName: string;
  terminalName: string;
  label: string;
  isActive: boolean;
  showCloseIcon?: boolean;
}>(), {
  showCloseIcon: false
});

const emit = defineEmits<{
  activate: [];
  close: [];
  connectionStatusChange: [status: TerminalConnectionStatus];
}>();

const isActive = computed(() => props.isActive);
const {
  focusTerminal,
  reloadTerminal,
  scrollTerminalToBottom,
  terminalElement
} = useProjectTerminalTab({
  projectName: props.projectName,
  terminalName: props.terminalName,
  isActive,
  onConnectionStatusChange: (status) => emit('connectionStatusChange', status)
});

const activate = () => {
  emit('activate');
};

const scrollToBottom = () => {
  void scrollTerminalToBottom();
};

const reload = () => {
  void reloadTerminal();
};

const close = () => {
  emit('close');
};

defineExpose({
  focusTerminal
});
</script>

<template>
  <section
    class="terminal-pane"
    :aria-label="`${label} terminal pane`"
    :data-active="String(isActive)"
    @focusin="activate"
    @pointerdown.capture="activate"
  >
    <TerminalHeader
      :label="label"
      :is-active="isActive"
      :show-close-icon="showCloseIcon"
      @scroll-to-bottom="scrollToBottom"
      @reload="reload"
      @close="close"
    />
    <section ref="terminalElement" class="terminal-screen" :aria-label="`${label} shell`"></section>
  </section>
</template>

<style scoped>
.terminal-pane {
  min-width: 0;
  min-height: 0;
  display: grid;
  grid-template-rows: 28px minmax(0, 1fr);
  overflow: hidden;
  background: var(--window);
}
</style>
