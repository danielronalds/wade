<script setup lang="ts">
import { GitBranch } from '@lucide/vue';
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import { RouterLink } from 'vue-router';
import { useProjectDetails } from '../composables/useProjectDetails';
import { useTerminalSession } from '../composables/useTerminalSession';

const props = defineProps<{
  projectName: string;
}>();

const terminalElement = ref<HTMLElement | null>(null);
const isConnectionStatusOpen = ref(true);
const {
  gitBranch,
  isLoading: isProjectDetailsLoading,
  loadProjectDetails
} = useProjectDetails(props.projectName);
const {
  connectionStatusText,
  isConnected,
  focusTerminal,
  start,
  stop
} = useTerminalSession(props.projectName, terminalElement);

const projectDisplayName = computed(() => props.projectName.split('-feature')[0] || props.projectName);
const connectionStatusToggleAction = computed(() => isConnectionStatusOpen.value ? 'hide' : 'show');
const connectionStatusLabel = computed(() => `${connectionStatusText.value}. Click to ${connectionStatusToggleAction.value} connection status text.`);
const gitBranchLabel = computed(() => {
  if (isProjectDetailsLoading.value) {
    return 'Loading branch';
  }

  return gitBranch.value || 'No branch';
});

const toggleConnectionStatusOpen = () => {
  isConnectionStatusOpen.value = !isConnectionStatusOpen.value;
  focusTerminal();
};

onMounted(() => {
  void loadProjectDetails();
  void start();
});

onBeforeUnmount(() => {
  stop();
});
</script>

<template>
  <section id="terminal-view" aria-label="Project terminal">
    <header id="terminal-topbar">
      <h1 id="project-summary">
        <RouterLink id="brand" :to="{ name: 'home' }">WADE</RouterLink>
        <span id="project-name" :title="projectName">{{ projectDisplayName }}</span>
        <span id="git-branch" :title="gitBranchLabel">
          <GitBranch :size="14" :stroke-width="1.75" aria-hidden="true" />
          <span>{{ gitBranchLabel }}</span>
        </span>
      </h1>
    </header>
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

#terminal-topbar {
  height: 42px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  padding: 0 16px;
  border-bottom: 1px solid var(--text);
  background: var(--window);
  color: var(--text);
  user-select: none;
}

#project-summary {
  min-width: 0;
  display: flex;
  margin: 0;
  align-items: center;
  gap: 14px;
}

#brand {
  margin: 0;
  color: var(--text);
  font-size: 24px;
  font-weight: 700;
  line-height: 1;
  text-decoration: none;
}

#brand:hover,
#brand:focus-visible {
  text-decoration: underline;
}

#project-name {
  flex: 0 1 auto;
  margin: 0;
  overflow: hidden;
  color: var(--muted);
  font-size: 14px;
  line-height: 1;
  text-overflow: ellipsis;
  white-space: nowrap;
}

#git-branch {
  min-width: 0;
  max-width: 45vw;
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 0;
  overflow: hidden;
  color: var(--text);
  font-size: 13px;
  line-height: 1;
  white-space: nowrap;
}

#git-branch svg {
  flex: 0 0 auto;
}

#git-branch span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

#connection-status {
  position: fixed;
  right: 8px;
  bottom: 8px;
  z-index: 2;
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
