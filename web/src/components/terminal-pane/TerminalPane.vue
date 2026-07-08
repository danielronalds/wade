<script setup lang="ts">
import { computed } from 'vue';
import { useTerminalPaneSession } from './composable/useTerminalPaneSession';
import TerminalHeader from './components/TerminalHeader.vue';
import type { TerminalConnectionStatus } from '../../types/terminalConnectionStatus';

const props = withDefaults(defineProps<{
  projectName: string;
  terminalName: string;
  label: string;
  isActive: boolean;
  showCloseIcon?: boolean;
  showZoomIcon?: boolean;
  isZoomed?: boolean;
  isCollapsed?: boolean;
}>(), {
  showCloseIcon: false,
  showZoomIcon: false,
  isZoomed: false,
  isCollapsed: false
});

const emit = defineEmits<{
  activate: [];
  close: [];
  connectionStatusChange: [status: TerminalConnectionStatus];
  toggleZoom: [];
}>();

const isActive = computed(() => props.isActive && !props.isCollapsed);
const {
  focusTerminal,
  reloadTerminal,
  scrollTerminalToBottom,
  terminalElement
} = useTerminalPaneSession({
  projectName: props.projectName,
  terminalName: props.terminalName,
  isActive,
  onConnectionStatusChange: (status) => emit('connectionStatusChange', status)
});

const activate = () => {
  if (props.isCollapsed) {
    return;
  }

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

const toggleZoom = () => {
  emit('toggleZoom');
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
    :data-collapsed="String(isCollapsed)"
    @focusin="activate"
    @pointerdown.capture="activate"
  >
    <TerminalHeader
      :label="label"
      :is-active="isActive"
      :show-close-icon="showCloseIcon"
      :show-zoom-icon="showZoomIcon"
      :is-zoomed="isZoomed"
      @scroll-to-bottom="scrollToBottom"
      @reload="reload"
      @close="close"
      @toggle-zoom="toggleZoom"
    />
    <section ref="terminalElement" class="terminal-screen" :aria-label="`${label} shell`"></section>
  </section>
</template>

<style scoped>
.terminal-pane {
  min-width: 0;
  min-height: 0;
  display: grid;
  grid-template-rows: var(--terminal-header-height, 28px) minmax(0, 1fr);
  overflow: hidden;
  background: var(--window);
}

.terminal-pane[data-collapsed="true"] {
  display: none;
}
</style>
