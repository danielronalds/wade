<script setup lang="ts">
import { FileSearch, FileTerminal, Server, SquareTerminal } from '@lucide/vue';
import type { Component } from 'vue';
import { WorkspaceTabs, type WorkspaceTab } from '@/types/workspaceTabs';

const props = defineProps<{
  activeTab: WorkspaceTab;
  isScratchpadOpen: boolean;
}>();

const emit = defineEmits<{
  selectTab: [tab: WorkspaceTab];
  toggleScratchpad: [];
}>();

const tabs: Array<{
  id: WorkspaceTab;
  label: string;
  icon: Component;
}> = [
  {
    id: WorkspaceTabs.Terminal,
    label: 'Terminal',
    icon: SquareTerminal
  },
  {
    id: WorkspaceTabs.Server,
    label: 'Server',
    icon: Server
  },
  {
    id: WorkspaceTabs.Review,
    label: 'Review',
    icon: FileSearch
  }
];

const selectTab = (tab: WorkspaceTab) => {
  emit('selectTab', tab);
};

const toggleScratchpad = () => {
  emit('toggleScratchpad');
};
</script>

<template>
  <aside id="workspace-sidebar" aria-label="Workspace sidebar">
    <nav aria-label="Workspace sections">
      <button
        v-for="tab in tabs"
        :key="tab.id"
        class="workspace-tab"
        type="button"
        :aria-label="tab.label"
        :aria-current="props.activeTab === tab.id ? 'page' : undefined"
        :data-active="String(props.activeTab === tab.id && !props.isScratchpadOpen)"
        :title="tab.label"
        @click="selectTab(tab.id)"
      >
        <component :is="tab.icon" :size="22" :stroke-width="1.6" aria-hidden="true" />
      </button>
      <button
        class="workspace-tab"
        type="button"
        aria-label="Scratchpad"
        :aria-pressed="props.isScratchpadOpen"
        :data-active="String(props.isScratchpadOpen)"
        title="Scratchpad"
        @click="toggleScratchpad"
      >
        <FileTerminal :size="22" :stroke-width="1.6" aria-hidden="true" />
      </button>
    </nav>
  </aside>
</template>

<style scoped>
#workspace-sidebar {
  width: 48px;
  height: 100%;
  border-right: 1px solid var(--text);
  background: var(--window);
  color: var(--text);
  user-select: none;
}

nav {
  display: flex;
  flex-direction: column;
  padding: 8px 0;
}

.workspace-tab {
  position: relative;
  width: 100%;
  height: 46px;
  display: grid;
  place-items: center;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--muted);
  cursor: pointer;
  font: inherit;
}

.workspace-tab::before {
  position: absolute;
  top: 8px;
  bottom: 8px;
  left: 0;
  width: 2px;
  background: transparent;
  content: '';
}

.workspace-tab:hover,
.workspace-tab:focus-visible,
.workspace-tab[data-active='true'] {
  color: var(--text);
}

.workspace-tab[data-active='true']::before {
  background: var(--text);
}
</style>
