<script setup lang="ts">
import { computed, nextTick, reactive, ref } from 'vue';
import ProjectSidebar from '../components/ProjectSidebar.vue';
import ReviewTab from '../components/ReviewTab.vue';
import ServerTab from '../components/ServerTab.vue';
import TerminalTab from '../components/TerminalTab.vue';
import TerminalTopbar from '../components/TerminalTopbar.vue';
import { useProjectKeyboardShortcuts } from '../composables/useProjectKeyboardShortcuts';
import { ProjectTabs, projectTabs, type ProjectTab } from '../types/projectTabs';
import {
  createDisconnectedTerminalConnectionStatus,
  type TerminalConnectionStatus
} from '../types/terminalConnectionStatus';

defineProps<{
  projectName: string;
}>();

type ProjectScreenComponent = {
  focusActiveTerminal: () => Promise<void>;
  switchToNextTerminal: () => Promise<void>;
};

const activeTab = ref<ProjectTab>(ProjectTabs.Terminal);
const terminalTab = ref<ProjectScreenComponent | null>(null);
const serverTab = ref<ProjectScreenComponent | null>(null);
const reviewTab = ref<ProjectScreenComponent | null>(null);
const connectionStatuses = reactive<Record<ProjectTab, TerminalConnectionStatus>>({
  [ProjectTabs.Terminal]: createDisconnectedTerminalConnectionStatus(),
  [ProjectTabs.Server]: createDisconnectedTerminalConnectionStatus(),
  [ProjectTabs.Review]: {
    connectionStatusText: 'Review',
    isConnected: true
  }
});
const activeConnectionStatus = computed(() => connectionStatuses[activeTab.value]);

const getActiveProjectScreen = () => {
  if (activeTab.value === ProjectTabs.Server) {
    return serverTab.value;
  }

  if (activeTab.value === ProjectTabs.Review) {
    return reviewTab.value;
  }

  return terminalTab.value;
};

const focusActiveProjectScreen = async () => {
  await nextTick();
  await getActiveProjectScreen()?.focusActiveTerminal();
};

const selectTab = (tab: ProjectTab) => {
  activeTab.value = tab;
  void focusActiveProjectScreen();
};

const selectTabBySlot = (slot: number) => {
  const tab = projectTabs[slot - 1];
  if (!tab) {
    return;
  }

  selectTab(tab);
};

const switchToNextTerminal = () => {
  void getActiveProjectScreen()?.switchToNextTerminal();
};

const selectTerminalTab = () => {
  selectTab(ProjectTabs.Terminal);
};

const updateConnectionStatus = (tab: ProjectTab, status: TerminalConnectionStatus) => {
  connectionStatuses[tab] = status;
};

useProjectKeyboardShortcuts({
  selectTabBySlot,
  switchToNextTerminal
});
</script>

<template>
  <section id="project-view" aria-label="Project">
    <TerminalTopbar
      :project-name="projectName"
      :connection-status-text="activeConnectionStatus.connectionStatusText"
      :is-connected="activeConnectionStatus.isConnected"
    />
    <section id="project-workspace">
      <ProjectSidebar :active-tab="activeTab" @select-tab="selectTab" />
      <section id="project-screens">
        <TerminalTab
          ref="terminalTab"
          v-show="activeTab === ProjectTabs.Terminal"
          :project-name="projectName"
          :is-active="activeTab === ProjectTabs.Terminal"
          @connection-status-change="updateConnectionStatus(ProjectTabs.Terminal, $event)"
        />
        <ServerTab
          ref="serverTab"
          v-show="activeTab === ProjectTabs.Server"
          :project-name="projectName"
          :is-active="activeTab === ProjectTabs.Server"
          @connection-status-change="updateConnectionStatus(ProjectTabs.Server, $event)"
        />
        <ReviewTab
          ref="reviewTab"
          v-show="activeTab === ProjectTabs.Review"
          :project-name="projectName"
          :is-active="activeTab === ProjectTabs.Review"
          @request-terminal-tab="selectTerminalTab"
        />
      </section>
    </section>
  </section>
</template>

<style scoped>
#project-view {
  width: 100vw;
  height: 100vh;
  display: grid;
  grid-template-rows: 42px minmax(0, 1fr);
  overflow: hidden;
  background: var(--window);
}

#project-workspace {
  min-width: 0;
  min-height: 0;
  display: grid;
  grid-template-columns: 48px minmax(0, 1fr);
  overflow: hidden;
}

#project-screens {
  min-width: 0;
  min-height: 0;
  overflow: hidden;
}
</style>
