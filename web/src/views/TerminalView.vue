<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import { useTerminalSession } from '../composables/useTerminalSession';

const props = defineProps<{
  projectName: string;
}>();

const terminalElement = ref<HTMLElement | null>(null);
const isConnectionStatusOpen = ref(true);
const {
  connectionStatusText,
  isConnected,
  focusTerminal,
  start,
  stop
} = useTerminalSession(props.projectName, terminalElement);

const connectionStatusToggleAction = computed(() => isConnectionStatusOpen.value ? 'hide' : 'show');
const connectionStatusLabel = computed(() => `${connectionStatusText.value}. Click to ${connectionStatusToggleAction.value} connection status text.`);

const toggleConnectionStatusOpen = () => {
  isConnectionStatusOpen.value = !isConnectionStatusOpen.value;
  focusTerminal();
};

onMounted(() => {
  void start();
});

onBeforeUnmount(() => {
  stop();
});
</script>

<template>
  <section id="terminal-view" aria-label="Project terminal">
    <header>
      <button
        id="connection-status"
        type="button"
        :aria-expanded="isConnectionStatusOpen"
        aria-live="polite"
        :aria-label="connectionStatusLabel"
        :data-connected="String(isConnected)"
        :data-open="String(isConnectionStatusOpen)"
        @click="toggleConnectionStatusOpen"
      >
        <span aria-hidden="true"></span>
        <span>{{ connectionStatusText }}</span>
      </button>
    </header>
    <section id="terminal" ref="terminalElement" aria-label="Interactive shell"></section>
  </section>
</template>

<style scoped>
#terminal-view {
  width: 100vw;
  height: 100vh;
  overflow: hidden;
  background: var(--window);
}

header {
  position: fixed;
  right: 8px;
  bottom: 8px;
  z-index: 1;
  user-select: none;
}

#connection-status {
  height: 24px;
  display: flex;
  align-items: center;
  gap: 7px;
  padding: 0 9px;
  border: 1px solid rgb(43 45 51 / 70%);
  border-radius: 999px;
  background: rgb(23 24 28 / 80%);
  color: var(--muted);
  font: inherit;
  font-size: 12px;
  letter-spacing: 0.01em;
  cursor: pointer;
  backdrop-filter: blur(10px);
}

#connection-status[data-open="false"] {
  width: 24px;
  justify-content: center;
  gap: 0;
  padding: 0;
}

#connection-status[data-open="false"] span:last-child {
  display: none;
}

#connection-status span:first-child {
  width: 7px;
  height: 7px;
  border-radius: 999px;
  background: var(--disconnected);
}

#connection-status[data-connected="true"] span:first-child {
  background: var(--connected);
}

#terminal {
  width: 100vw;
  height: 100vh;
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
