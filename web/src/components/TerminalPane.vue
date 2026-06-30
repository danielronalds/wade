<script setup lang="ts">
import { RefreshCw } from '@lucide/vue';
import { computed } from 'vue';
import { useProjectTerminalTab } from '../composables/useProjectTerminalTab';
import type { TerminalConnectionStatus } from '../types/terminalConnectionStatus';

const props = defineProps<{
  projectName: string;
  terminalName: string;
  label: string;
  isActive: boolean;
}>();

const emit = defineEmits<{
  activate: [];
  connectionStatusChange: [status: TerminalConnectionStatus];
}>();

const isActive = computed(() => props.isActive);
const { reloadTerminal, terminalElement } = useProjectTerminalTab({
  projectName: props.projectName,
  terminalName: props.terminalName,
  isActive,
  onConnectionStatusChange: (status) => emit('connectionStatusChange', status)
});

const activate = () => {
  emit('activate');
};

const reload = () => {
  void reloadTerminal();
};
</script>

<template>
  <section
    class="terminal-pane"
    :aria-label="`${label} terminal pane`"
    :data-active="String(isActive)"
    @focusin="activate"
    @pointerdown.capture="activate"
  >
    <header class="terminal-pane-header">
      <h2>{{ label }}</h2>
      <button
        class="reload-terminal"
        type="button"
        :aria-label="`Reload ${label} terminal`"
        :title="`Reload ${label} terminal`"
        @click.stop="reload"
      >
        <RefreshCw :size="14" :stroke-width="1.7" aria-hidden="true" />
      </button>
    </header>
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

.terminal-pane-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 0 8px 0 12px;
  border-bottom: 1px solid rgb(248 248 242 / 45%);
  color: var(--muted);
  user-select: none;
}

.terminal-pane[data-active="true"] .terminal-pane-header {
  color: var(--text);
}

h2 {
  margin: 0;
  font-size: 12px;
  font-weight: 400;
  line-height: 1;
}

.reload-terminal {
  width: 22px;
  height: 22px;
  display: grid;
  place-items: center;
  padding: 0;
  border: 0;
  background: transparent;
  color: inherit;
  cursor: pointer;
}

.reload-terminal:hover,
.reload-terminal:focus-visible {
  color: var(--text);
}
</style>
