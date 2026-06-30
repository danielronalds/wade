<script setup lang="ts">
import { computed, reactive, ref } from 'vue';
import ProjectSidebar from '../components/ProjectSidebar.vue';
import ServerTab from '../components/ServerTab.vue';
import TerminalTab from '../components/TerminalTab.vue';
import TerminalTopbar from '../components/TerminalTopbar.vue';
import { ProjectTabs, type ProjectTab } from '../projectTabs';
import {
  createDisconnectedTerminalConnectionStatus,
  type TerminalConnectionStatus
} from '../terminalConnectionStatus';

defineProps<{
  projectName: string;
}>();

const activeTab = ref<ProjectTab>(ProjectTabs.Terminal);
const connectionStatuses = reactive<Record<ProjectTab, TerminalConnectionStatus>>({
  [ProjectTabs.Terminal]: createDisconnectedTerminalConnectionStatus(),
  [ProjectTabs.Server]: createDisconnectedTerminalConnectionStatus()
});
const activeConnectionStatus = computed(() => connectionStatuses[activeTab.value]);

const selectTab = (tab: ProjectTab) => {
  activeTab.value = tab;
};

const updateConnectionStatus = (tab: ProjectTab, status: TerminalConnectionStatus) => {
  connectionStatuses[tab] = status;
};
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
          v-show="activeTab === ProjectTabs.Terminal"
          :project-name="projectName"
          :is-active="activeTab === ProjectTabs.Terminal"
          @connection-status-change="updateConnectionStatus(ProjectTabs.Terminal, $event)"
        />
        <ServerTab
          v-show="activeTab === ProjectTabs.Server"
          :project-name="projectName"
          :is-active="activeTab === ProjectTabs.Server"
          @connection-status-change="updateConnectionStatus(ProjectTabs.Server, $event)"
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
