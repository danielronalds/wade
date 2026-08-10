<script setup lang="ts">
import { storeToRefs } from 'pinia';
import { computed, nextTick, onMounted, reactive, ref, toRef, watch } from 'vue';
import { useSettingsStore } from '@/stores/useSettingsStore';
import { useWorkspaceSessionStore } from '@/stores/useWorkspaceSessionStore';
import { defaultAgents, type Agent } from '@/types/settings';
import { useTerminalTabPaneZoom } from '@/views/workspace/tabs/terminal/composables/useTerminalTabPaneZoom';
import { TerminalPanes, type TerminalPaneId } from '@/types/terminalPanes';
import {
  combineTerminalConnectionStatuses,
  createDisconnectedTerminalConnectionStatus,
  type TerminalConnectionStatus
} from '@/types/terminalConnectionStatus';
import CollapsedTerminalRail from '@/views/workspace/tabs/terminal/components/CollapsedTerminalRail.vue';
import TerminalPane from '@/views/workspace/components/terminal-pane/TerminalPane.vue';

const props = defineProps<{
  workspaceId: string;
  isActive: boolean;
}>();

const emit = defineEmits<{
  connectionStatusChange: [status: TerminalConnectionStatus];
}>();

const { loadSettings } = useSettingsStore();
const workspaceSessionStore = useWorkspaceSessionStore();
const {
  selectedAgentName,
  terminalActivePane: activePane,
  terminalZoomedPane: zoomedPane
} = storeToRefs(workspaceSessionStore);

type TerminalPaneComponent = {
  focusTerminal: () => Promise<void>;
};

const agents = ref<Agent[]>([]);
const openedAgentNames = ref<string[]>([]);
const miscPane = ref<TerminalPaneComponent | null>(null);
const agentPanes = new Map<string, TerminalPaneComponent>();
const agentConnectionStatuses = reactive<Record<string, TerminalConnectionStatus>>({});
const miscConnectionStatus = ref<TerminalConnectionStatus>(createDisconnectedTerminalConnectionStatus());
const openedAgents = computed(() =>
  openedAgentNames.value
    .map((agentName) => agents.value.find((agent) => agent.name === agentName))
    .filter((agent): agent is Agent => Boolean(agent))
);
const selectedAgentConnectionStatus = computed(
  () => agentConnectionStatuses[selectedAgentName.value] ?? createDisconnectedTerminalConnectionStatus()
);
const combinedConnectionStatus = computed(() =>
  combineTerminalConnectionStatuses([selectedAgentConnectionStatus.value, miscConnectionStatus.value])
);

const getPaneComponent = (pane: TerminalPaneId) =>
  pane === TerminalPanes.Misc ? miscPane.value : agentPanes.get(selectedAgentName.value);

const focusPane = async (pane: TerminalPaneId) => {
  await getPaneComponent(pane)?.focusTerminal();
};

const {
  activatePane,
  focusActiveTerminal,
  focusFirstPane,
  isAgentPaneActive,
  isAgentPaneCollapsed,
  isAgentPaneZoomed,
  isMiscPaneActive,
  isMiscPaneCollapsed,
  isMiscPaneZoomed,
  restoreSplitView,
  switchToNextTerminal,
  terminalTabLayout,
  toggleActivePaneZoom,
  togglePaneZoom
} = useTerminalTabPaneZoom({
  activePane,
  zoomedPane,
  isActive: toRef(props, 'isActive'),
  focusPane
});

const updateAgentConnectionStatus = (agentName: string, status: TerminalConnectionStatus) => {
  agentConnectionStatuses[agentName] = status;
};

const updateMiscConnectionStatus = (status: TerminalConnectionStatus) => {
  miscConnectionStatus.value = status;
};

const isTerminalPaneComponent = (pane: unknown): pane is TerminalPaneComponent =>
  Boolean(pane && typeof pane === 'object' && 'focusTerminal' in pane);

const setAgentPane = (agentName: string, pane: unknown) => {
  if (isTerminalPaneComponent(pane)) {
    agentPanes.set(agentName, pane);
    return;
  }

  agentPanes.delete(agentName);
};

const focusSelectedAgentTerminal = async () => {
  if (!props.isActive) {
    return;
  }

  await nextTick();
  await agentPanes.get(selectedAgentName.value)?.focusTerminal();
};

const selectAgent = async (agentName: string) => {
  if (!agents.value.some((agent) => agent.name === agentName)) {
    return;
  }

  selectedAgentName.value = agentName;
  activatePane(TerminalPanes.Agent);
  if (!openedAgentNames.value.includes(agentName)) {
    openedAgentNames.value = [...openedAgentNames.value, agentName];
  }

  await focusSelectedAgentTerminal();
};

const loadAgents = async () => {
  try {
    const settings = await loadSettings();
    agents.value = settings.agents.map((agent) => ({ ...agent }));
  } catch {
    agents.value = defaultAgents.map((agent) => ({ ...agent }));
  }
};

watch(
  agents,
  (nextAgents) => {
    if (nextAgents.length === 0) {
      return;
    }

    if (!nextAgents.some((agent) => agent.name === selectedAgentName.value)) {
      const defaultAgent = nextAgents.find((agent) => agent.default);
      selectedAgentName.value = defaultAgent?.name ?? nextAgents[0].name;
    }

    if (!openedAgentNames.value.includes(selectedAgentName.value)) {
      openedAgentNames.value = [...openedAgentNames.value, selectedAgentName.value];
    }

    if (isAgentPaneActive.value) {
      void focusSelectedAgentTerminal();
    }
  },
  { immediate: true }
);

watch(
  combinedConnectionStatus,
  (status) => {
    emit('connectionStatusChange', status);
  },
  { immediate: true }
);

onMounted(() => {
  void loadAgents();
});

defineExpose({
  focusActiveTerminal,
  focusFirstPane,
  switchToNextTerminal,
  toggleActivePaneZoom
});
</script>

<template>
  <section id="terminal-tab" :data-layout="terminalTabLayout" aria-label="Terminal screens">
    <CollapsedTerminalRail v-if="isAgentPaneCollapsed" label="Agent" side="left" @restore="restoreSplitView" />
    <TerminalPane
      v-for="agent in openedAgents"
      :key="agent.name"
      :ref="(pane) => setAgentPane(agent.name, pane)"
      v-show="selectedAgentName === agent.name"
      class="agent-pane"
      :class="{ 'agent-pane--split': terminalTabLayout === 'split' }"
      :workspace-id="workspaceId"
      :terminal-id="`agent:${agent.name.toLowerCase()}`"
      :agent-name="agent.name"
      :agents="agents"
      :selected-agent-name="selectedAgentName"
      label="Agent"
      :is-active="isAgentPaneActive && selectedAgentName === agent.name"
      :is-collapsed="isAgentPaneCollapsed"
      :is-zoomed="isAgentPaneZoomed"
      lazy
      show-zoom-icon
      @activate="activatePane(TerminalPanes.Agent)"
      @agent-change="selectAgent"
      @toggle-zoom="togglePaneZoom(TerminalPanes.Agent)"
      @connection-status-change="updateAgentConnectionStatus(agent.name, $event)"
    />
    <TerminalPane
      ref="miscPane"
      class="misc-pane"
      :workspace-id="workspaceId"
      :terminal-id="TerminalPanes.Misc"
      label="Misc"
      :is-active="isMiscPaneActive"
      :is-collapsed="isMiscPaneCollapsed"
      :is-zoomed="isMiscPaneZoomed"
      show-zoom-icon
      @activate="activatePane(TerminalPanes.Misc)"
      @toggle-zoom="togglePaneZoom(TerminalPanes.Misc)"
      @connection-status-change="updateMiscConnectionStatus"
    />
    <CollapsedTerminalRail v-if="isMiscPaneCollapsed" label="Misc" side="right" @restore="restoreSplitView" />
  </section>
</template>

<style scoped>
#terminal-tab {
  --terminal-header-height: 28px;

  width: 100%;
  height: 100%;
  min-width: 0;
  min-height: 0;
  display: grid;
  grid-template-columns: 80ch minmax(0, 1fr);
  grid-template-areas: 'agent misc';
  overflow: hidden;
}

#terminal-tab[data-layout='agent-zoomed'] {
  grid-template-columns: minmax(0, 1fr) var(--terminal-header-height);
}

#terminal-tab[data-layout='misc-zoomed'] {
  grid-template-columns: var(--terminal-header-height) minmax(0, 1fr);
}

.agent-pane {
  grid-area: agent;
}

.agent-pane--split {
  border-right: 1px solid var(--text);
}

.misc-pane {
  grid-area: misc;
}
</style>
