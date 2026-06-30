<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue';
import TerminalTopbar from '../components/TerminalTopbar.vue';
import { useTerminalSession } from '../composables/useTerminalSession';

const props = defineProps<{
  projectName: string;
}>();

const terminalElement = ref<HTMLElement | null>(null);
const {
  connectionStatusText,
  isConnected,
  start,
  stop
} = useTerminalSession(props.projectName, terminalElement);

onMounted(() => {
  void start();
});

onBeforeUnmount(() => {
  stop();
});
</script>

<template>
  <section id="terminal-view" aria-label="Project terminal">
    <TerminalTopbar
      :project-name="projectName"
      :connection-status-text="connectionStatusText"
      :is-connected="isConnected"
    />
    <section id="terminal" ref="terminalElement" aria-label="Interactive shell"></section>
  </section>
</template>

<style scoped>
#terminal-view {
  width: 100vw;
  height: 100vh;
  display: grid;
  grid-template-rows: 42px minmax(0, 1fr);
  overflow: hidden;
  background: var(--window);
}

#terminal {
  width: 100%;
  height: 100%;
  min-height: 0;
  padding: 10px 12px 14px;
  overflow: hidden;
}

#terminal :deep(.xterm) {
  height: 100%;
  padding: 2px;
}

#terminal :deep(.xterm-viewport) {
  scrollbar-width: thin;
}
</style>
