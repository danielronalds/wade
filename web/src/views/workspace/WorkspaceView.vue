<script setup lang="ts">
import { storeToRefs } from 'pinia';
import { computed, nextTick, onMounted, reactive, ref } from 'vue';
import { useWorkspaceDetailsStore } from '@/stores/useWorkspaceDetailsStore';
import { useWorkspaceSessionStore } from '@/stores/useWorkspaceSessionStore';
import type { ReviewScreenComponent, WorkspaceScreenComponent } from '@/types/workspaceScreens';
import { WorkspaceTabs, workspaceTabs, type WorkspaceTab } from '@/types/workspaceTabs';
import {
  createDisconnectedTerminalConnectionStatus,
  type TerminalConnectionStatus
} from '@/types/terminalConnectionStatus';
import ScratchpadTerminal from '@/views/workspace/components/ScratchpadTerminal.vue';
import WorkspaceSidebar from '@/views/workspace/components/WorkspaceSidebar.vue';
import WorkspaceTopbar from '@/views/workspace/components/WorkspaceTopbar.vue';
import { useWorkspaceEventHandlers } from '@/views/workspace/composables/useWorkspaceEventHandlers';
import { useWorkspaceKeyboardShortcuts } from '@/views/workspace/composables/useWorkspaceKeyboardShortcuts';
import ReviewTab from '@/views/workspace/tabs/review/ReviewTab.vue';
import ServerTab from '@/views/workspace/tabs/server/ServerTab.vue';
import TerminalTab from '@/views/workspace/tabs/terminal/TerminalTab.vue';

const props = defineProps<{
  workspaceId: string;
}>();

const workspaceDetailsStore = useWorkspaceDetailsStore();
const workspaceSessionStore = useWorkspaceSessionStore();
workspaceSessionStore.activateWorkspaceSession(props.workspaceId);
const { activeTab, isScratchpadOpen } = storeToRefs(workspaceSessionStore);

const terminalTab = ref<WorkspaceScreenComponent | null>(null);
const serverTab = ref<WorkspaceScreenComponent | null>(null);
const reviewTab = ref<ReviewScreenComponent | null>(null);
const scratchpadTerminal = ref<WorkspaceScreenComponent | null>(null);
const hasScratchpadOpened = ref(isScratchpadOpen.value);
const connectionStatuses = reactive<Record<WorkspaceTab, TerminalConnectionStatus>>({
  [WorkspaceTabs.Terminal]: createDisconnectedTerminalConnectionStatus(),
  [WorkspaceTabs.Server]: createDisconnectedTerminalConnectionStatus(),
  [WorkspaceTabs.Review]: {
    connectionStatusText: 'Review',
    isConnected: true
  }
});
const scratchpadConnectionStatus = ref<TerminalConnectionStatus>(createDisconnectedTerminalConnectionStatus());
const isWorkspaceScreenActive = computed(() => !isScratchpadOpen.value);
const isTerminalTabActive = computed(() => isWorkspaceScreenActive.value && activeTab.value === WorkspaceTabs.Terminal);
const isServerTabActive = computed(() => isWorkspaceScreenActive.value && activeTab.value === WorkspaceTabs.Server);
const isReviewTabActive = computed(() => isWorkspaceScreenActive.value && activeTab.value === WorkspaceTabs.Review);
const activeConnectionStatus = computed(() =>
  isScratchpadOpen.value ? scratchpadConnectionStatus.value : connectionStatuses[activeTab.value]
);

const getActiveWorkspaceScreen = () => {
  if (activeTab.value === WorkspaceTabs.Server) {
    return serverTab.value;
  }

  if (activeTab.value === WorkspaceTabs.Review) {
    return reviewTab.value;
  }

  return terminalTab.value;
};

const focusActiveWorkspaceScreen = async () => {
  await nextTick();
  await getActiveWorkspaceScreen()?.focusActiveTerminal();
};

const selectTab = (tab: WorkspaceTab) => {
  activeTab.value = tab;
  void focusActiveWorkspaceScreen();
};

const scratchpadSidebarSlot = workspaceTabs.length + 1;

const selectSidebarItemBySlot = (slot: number) => {
  if (slot === scratchpadSidebarSlot) {
    void openScratchpadTerminal();
    return;
  }

  const tab = workspaceTabs[slot - 1];
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

  void getActiveWorkspaceScreen()?.switchToNextTerminal();
};

const toggleTerminalZoom = () => {
  if (isScratchpadOpen.value || activeTab.value !== WorkspaceTabs.Terminal) {
    return;
  }

  void terminalTab.value?.toggleActivePaneZoom?.();
};

const selectFirstTerminalPane = async () => {
  activeTab.value = WorkspaceTabs.Terminal;
  await nextTick();

  if (terminalTab.value?.focusFirstPane) {
    await terminalTab.value.focusFirstPane();
    return;
  }

  await terminalTab.value?.focusActiveTerminal();
};

const updateConnectionStatus = (tab: WorkspaceTab, status: TerminalConnectionStatus) => {
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
  await focusActiveWorkspaceScreen();
};

const handleScratchpadTerminalEnd = async () => {
  isScratchpadOpen.value = false;
  hasScratchpadOpened.value = false;
  scratchpadConnectionStatus.value = createDisconnectedTerminalConnectionStatus();

  await nextTick();
  await focusActiveWorkspaceScreen();
};

const toggleScratchpadTerminal = () => {
  if (isScratchpadOpen.value) {
    void closeScratchpadTerminal();
    return;
  }

  void openScratchpadTerminal();
};

const startReviewFromCommandPalette = async () => {
  activeTab.value = WorkspaceTabs.Review;
  await nextTick();
  await reviewTab.value?.startReview();
  await reviewTab.value?.focusActiveTerminal();
};

const cancelReviewFromCommandPalette = async () => {
  await reviewTab.value?.cancelReview();
};

useWorkspaceEventHandlers({
  cancelReview: cancelReviewFromCommandPalette,
  getWorkspaceId: () => props.workspaceId,
  startReview: startReviewFromCommandPalette
});

useWorkspaceKeyboardShortcuts({
  selectSidebarItemBySlot,
  switchToNextTerminal,
  toggleScratchpadTerminal,
  toggleTerminalZoom
});

onMounted(async () => {
  void workspaceDetailsStore.loadWorkspaceDetails(props.workspaceId);

  if (!isScratchpadOpen.value) {
    return;
  }

  await nextTick();
  await scratchpadTerminal.value?.focusActiveTerminal();
});
</script>

<template>
  <section id="workspace-view" aria-label="Workspace">
    <WorkspaceTopbar
      :workspace-id="workspaceId"
      :connection-status-text="activeConnectionStatus.connectionStatusText"
      :is-connected="activeConnectionStatus.isConnected"
    />
    <section id="workspace-layout">
      <WorkspaceSidebar
        :active-tab="activeTab"
        :is-scratchpad-open="isScratchpadOpen"
        @select-tab="selectTab"
        @toggle-scratchpad="toggleScratchpadTerminal"
      />
      <section id="workspace-screens">
        <TerminalTab
          ref="terminalTab"
          v-show="activeTab === WorkspaceTabs.Terminal"
          :workspace-id="workspaceId"
          :is-active="isTerminalTabActive"
          @connection-status-change="updateConnectionStatus(WorkspaceTabs.Terminal, $event)"
        />
        <ServerTab
          ref="serverTab"
          v-show="activeTab === WorkspaceTabs.Server"
          :workspace-id="workspaceId"
          :is-active="isServerTabActive"
          @connection-status-change="updateConnectionStatus(WorkspaceTabs.Server, $event)"
        />
        <ReviewTab
          ref="reviewTab"
          v-show="activeTab === WorkspaceTabs.Review"
          :workspace-id="workspaceId"
          :is-active="isReviewTabActive"
          @request-terminal-tab="selectFirstTerminalPane"
        />
      </section>
    </section>
    <ScratchpadTerminal
      v-if="hasScratchpadOpened"
      ref="scratchpadTerminal"
      :workspace-id="workspaceId"
      :is-open="isScratchpadOpen"
      :is-active="isScratchpadOpen"
      @close="closeScratchpadTerminal"
      @connection-status-change="updateScratchpadConnectionStatus"
      @terminal-end="handleScratchpadTerminalEnd"
    />
  </section>
</template>

<style scoped>
#workspace-view {
  width: 100vw;
  height: 100vh;
  display: grid;
  grid-template-rows: 42px minmax(0, 1fr);
  overflow: hidden;
  background: var(--window);
}

#workspace-layout {
  min-width: 0;
  min-height: 0;
  display: grid;
  grid-template-columns: 48px minmax(0, 1fr);
  overflow: hidden;
}

#workspace-screens {
  min-width: 0;
  min-height: 0;
  overflow: hidden;
}
</style>
