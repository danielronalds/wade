<script setup lang="ts">
import { computed, nextTick, onMounted, reactive, ref } from 'vue';
import ProjectSidebar from '@/views/project/components/ProjectSidebar.vue';
import ReviewTab from '@/views/project/tabs/review/ReviewTab.vue';
import ScratchpadTerminal from '@/views/project/components/ScratchpadTerminal.vue';
import ServerTab from '@/views/project/tabs/server/ServerTab.vue';
import TerminalTab from '@/views/project/tabs/terminal/TerminalTab.vue';
import ProjectTopbar from '@/views/project/components/ProjectTopbar.vue';
import { useProjectEventHandlers } from '@/views/project/composables/useProjectEventHandlers';
import { useProjectKeyboardShortcuts } from '@/views/project/composables/useProjectKeyboardShortcuts';
import { useProjectDetailsStore } from '@/stores/useProjectDetailsStore';
import type { ProjectScreenComponent, ReviewScreenComponent } from '@/types/projectScreens';
import { ProjectTabs, projectTabs, type ProjectTab } from '@/types/projectTabs';
import {
  createDisconnectedTerminalConnectionStatus,
  type TerminalConnectionStatus
} from '@/types/terminalConnectionStatus';

const props = defineProps<{
  projectName: string;
}>();

const projectDetailsStore = useProjectDetailsStore();

const activeTab = ref<ProjectTab>(ProjectTabs.Terminal);
const terminalTab = ref<ProjectScreenComponent | null>(null);
const serverTab = ref<ProjectScreenComponent | null>(null);
const reviewTab = ref<ReviewScreenComponent | null>(null);
const scratchpadTerminal = ref<ProjectScreenComponent | null>(null);
const isScratchpadOpen = ref(false);
const hasScratchpadOpened = ref(false);
const connectionStatuses = reactive<Record<ProjectTab, TerminalConnectionStatus>>({
  [ProjectTabs.Terminal]: createDisconnectedTerminalConnectionStatus(),
  [ProjectTabs.Server]: createDisconnectedTerminalConnectionStatus(),
  [ProjectTabs.Review]: {
    connectionStatusText: 'Review',
    isConnected: true
  }
});
const scratchpadConnectionStatus = ref<TerminalConnectionStatus>(createDisconnectedTerminalConnectionStatus());
const isProjectScreenActive = computed(() => !isScratchpadOpen.value);
const isTerminalTabActive = computed(() => isProjectScreenActive.value && activeTab.value === ProjectTabs.Terminal);
const isServerTabActive = computed(() => isProjectScreenActive.value && activeTab.value === ProjectTabs.Server);
const isReviewTabActive = computed(() => isProjectScreenActive.value && activeTab.value === ProjectTabs.Review);
const activeConnectionStatus = computed(() => isScratchpadOpen.value
  ? scratchpadConnectionStatus.value
  : connectionStatuses[activeTab.value]);

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

const scratchpadSidebarSlot = projectTabs.length + 1;

const selectSidebarItemBySlot = (slot: number) => {
  if (slot === scratchpadSidebarSlot) {
    void openScratchpadTerminal();
    return;
  }

  const tab = projectTabs[slot - 1];
  if (!tab) {
    return;
  }

  selectTab(tab);
};

const switchToNextTerminal = () => {
  if (isScratchpadOpen.value) {
    void scratchpadTerminal.value?.switchToNextTerminal();
    return;
  }

  void getActiveProjectScreen()?.switchToNextTerminal();
};

const toggleTerminalZoom = () => {
  if (isScratchpadOpen.value || activeTab.value !== ProjectTabs.Terminal) {
    return;
  }

  void terminalTab.value?.toggleActivePaneZoom?.();
};

const selectFirstTerminalPane = async () => {
  activeTab.value = ProjectTabs.Terminal;
  await nextTick();

  if (terminalTab.value?.focusFirstPane) {
    await terminalTab.value.focusFirstPane();
    return;
  }

  await terminalTab.value?.focusActiveTerminal();
};

const updateConnectionStatus = (tab: ProjectTab, status: TerminalConnectionStatus) => {
  connectionStatuses[tab] = status;
};

const updateScratchpadConnectionStatus = (status: TerminalConnectionStatus) => {
  scratchpadConnectionStatus.value = status;
};

const openScratchpadTerminal = async () => {
  hasScratchpadOpened.value = true;
  isScratchpadOpen.value = true;
  await nextTick();
  await scratchpadTerminal.value?.focusActiveTerminal();
};

const closeScratchpadTerminal = async () => {
  isScratchpadOpen.value = false;
  await focusActiveProjectScreen();
};

const handleScratchpadSessionEnd = async () => {
  isScratchpadOpen.value = false;
  hasScratchpadOpened.value = false;
  scratchpadConnectionStatus.value = createDisconnectedTerminalConnectionStatus();

  await nextTick();
  await focusActiveProjectScreen();
};

const toggleScratchpadTerminal = () => {
  if (isScratchpadOpen.value) {
    void closeScratchpadTerminal();
    return;
  }

  void openScratchpadTerminal();
};

const startReviewFromCommandPalette = async () => {
  activeTab.value = ProjectTabs.Review;
  await nextTick();
  await reviewTab.value?.startReview();
  await reviewTab.value?.focusActiveTerminal();
};

const cancelReviewFromCommandPalette = async () => {
  await reviewTab.value?.cancelReview();
};

useProjectEventHandlers({
  cancelReview: cancelReviewFromCommandPalette,
  getProjectName: () => props.projectName,
  startReview: startReviewFromCommandPalette
});

useProjectKeyboardShortcuts({
  selectSidebarItemBySlot,
  switchToNextTerminal,
  toggleScratchpadTerminal,
  toggleTerminalZoom
});

onMounted(() => {
  void projectDetailsStore.loadProjectDetails(props.projectName);
});
</script>

<template>
  <section id="project-view" aria-label="Project">
    <ProjectTopbar
      :project-name="projectName"
      :connection-status-text="activeConnectionStatus.connectionStatusText"
      :is-connected="activeConnectionStatus.isConnected"
    />
    <section id="project-workspace">
      <ProjectSidebar
        :active-tab="activeTab"
        :is-scratchpad-open="isScratchpadOpen"
        @select-tab="selectTab"
        @toggle-scratchpad="toggleScratchpadTerminal"
      />
      <section id="project-screens">
        <TerminalTab
          ref="terminalTab"
          v-show="activeTab === ProjectTabs.Terminal"
          :project-name="projectName"
          :is-active="isTerminalTabActive"
          @connection-status-change="updateConnectionStatus(ProjectTabs.Terminal, $event)"
        />
        <ServerTab
          ref="serverTab"
          v-show="activeTab === ProjectTabs.Server"
          :project-name="projectName"
          :is-active="isServerTabActive"
          @connection-status-change="updateConnectionStatus(ProjectTabs.Server, $event)"
        />
        <ReviewTab
          ref="reviewTab"
          v-show="activeTab === ProjectTabs.Review"
          :project-name="projectName"
          :is-active="isReviewTabActive"
          @request-terminal-tab="selectFirstTerminalPane"
        />
      </section>
    </section>
    <ScratchpadTerminal
      v-if="hasScratchpadOpened"
      ref="scratchpadTerminal"
      :project-name="projectName"
      :is-open="isScratchpadOpen"
      :is-active="isScratchpadOpen"
      @close="closeScratchpadTerminal"
      @connection-status-change="updateScratchpadConnectionStatus"
      @session-end="handleScratchpadSessionEnd"
    />
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
