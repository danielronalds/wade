import { defineStore } from 'pinia';
import { computed, reactive, ref, watch } from 'vue';
import { listWorkspaceTerminals, TerminalStatus } from '@/api/generated/wade';
import { TerminalPanes, terminalPanes, type TerminalPaneId } from '@/types/terminalPanes';
import { WorkspaceTabs, workspaceTabs, type WorkspaceTab } from '@/types/workspaceTabs';

export type WorkspaceSessionCheckpoint = {
  activeTab: WorkspaceTab;
  isScratchpadOpen: boolean;
  terminal: {
    activePane: TerminalPaneId;
    zoomedPane: TerminalPaneId | null;
    selectedAgentName: string;
  };
  review: null;
};

type WorkspaceSessionEntry = {
  state: WorkspaceSessionCheckpoint;
  isInitialised: boolean;
  lastSerialisedCheckpoint: string;
};

export const useWorkspaceSessionStore = defineStore('workspace-session', () => {
  const workspaceSessions = new Map<string, WorkspaceSessionEntry>();
  const preparationRequests = new Map<string, Promise<void>>();
  const activeWorkspaceId = ref('');

  const activeTab = computed<WorkspaceTab>({
    get: () => workspaceSessions.get(activeWorkspaceId.value)?.state.activeTab ?? WorkspaceTabs.Terminal,
    set: (activeTab) => {
      const entry = workspaceSessions.get(activeWorkspaceId.value);
      if (entry) {
        entry.state.activeTab = activeTab;
      }
    }
  });
  const isScratchpadOpen = computed({
    get: () => workspaceSessions.get(activeWorkspaceId.value)?.state.isScratchpadOpen ?? false,
    set: (isScratchpadOpen) => {
      const entry = workspaceSessions.get(activeWorkspaceId.value);
      if (entry) {
        entry.state.isScratchpadOpen = isScratchpadOpen;
      }
    }
  });
  const terminalActivePane = computed<TerminalPaneId>({
    get: () => workspaceSessions.get(activeWorkspaceId.value)?.state.terminal.activePane ?? TerminalPanes.Agent,
    set: (activePane) => {
      const entry = workspaceSessions.get(activeWorkspaceId.value);
      if (entry) {
        entry.state.terminal.activePane = activePane;
      }
    }
  });
  const terminalZoomedPane = computed<TerminalPaneId | null>({
    get: () => workspaceSessions.get(activeWorkspaceId.value)?.state.terminal.zoomedPane ?? null,
    set: (zoomedPane) => {
      const entry = workspaceSessions.get(activeWorkspaceId.value);
      if (entry) {
        entry.state.terminal.zoomedPane = zoomedPane;
      }
    }
  });
  const selectedAgentName = computed({
    get: () => workspaceSessions.get(activeWorkspaceId.value)?.state.terminal.selectedAgentName ?? '',
    set: (selectedAgentName) => {
      const entry = workspaceSessions.get(activeWorkspaceId.value);
      if (entry) {
        entry.state.terminal.selectedAgentName = selectedAgentName;
      }
    }
  });

  const activateWorkspaceSession = (workspaceId: string) => {
    ensureWorkspaceSession(workspaceId);
    activeWorkspaceId.value = workspaceId;
  };

  const resetWorkspaceSession = (workspaceId: string) => {
    const state = ensureWorkspaceSession(workspaceId);
    const entry = workspaceSessions.get(workspaceId)!;
    const freshState = createFreshWorkspaceSession();

    entry.isInitialised = false;
    replaceWorkspaceSession(state, freshState);
    removeStoredValue(workspaceSessionStorageKey(workspaceId));
    entry.lastSerialisedCheckpoint = serialiseCheckpoint(state);
    entry.isInitialised = true;
  };

  const prepareWorkspaceSession = (workspaceId: string) => {
    const existingRequest = preparationRequests.get(workspaceId);
    if (existingRequest) {
      return existingRequest;
    }

    const preparationRequest = (async () => {
      try {
        const terminalList = await listWorkspaceTerminals(workspaceId);
        const hasRunningTerminals = terminalList.items.some((terminal) => (
          terminal.status === TerminalStatus.TerminalStatusRunning
        ));

        if (!hasRunningTerminals) {
          resetWorkspaceSession(workspaceId);
          return;
        }

        hydrateWorkspaceSession(workspaceId);
      } catch {
        // Failed validation must not overwrite a checkpoint while daemon state is unknown.
        ensureWorkspaceSession(workspaceId);
      } finally {
        preparationRequests.delete(workspaceId);
      }
    })();

    preparationRequests.set(workspaceId, preparationRequest);
    return preparationRequest;
  };

  const getSelectedAgentName = (workspaceId: string) => (
    ensureWorkspaceSession(workspaceId).terminal.selectedAgentName
  );

  const ensureWorkspaceSession = (workspaceId: string) => {
    const existingEntry = workspaceSessions.get(workspaceId);
    if (existingEntry) {
      return existingEntry.state;
    }

    const state = reactive<WorkspaceSessionCheckpoint>(createFreshWorkspaceSession());
    const entry: WorkspaceSessionEntry = {
      state,
      isInitialised: false,
      lastSerialisedCheckpoint: serialiseCheckpoint(state)
    };

    watch(state, () => {
      if (!entry.isInitialised) {
        return;
      }

      const serialisedCheckpoint = serialiseCheckpoint(state);
      if (serialisedCheckpoint === entry.lastSerialisedCheckpoint) {
        return;
      }

      storeValue(workspaceSessionStorageKey(workspaceId), serialisedCheckpoint);
      entry.lastSerialisedCheckpoint = serialisedCheckpoint;
    }, { deep: true });

    workspaceSessions.set(workspaceId, entry);
    return state;
  };

  const hydrateWorkspaceSession = (workspaceId: string) => {
    const state = ensureWorkspaceSession(workspaceId);
    const entry = workspaceSessions.get(workspaceId)!;
    if (entry.isInitialised) {
      return;
    }

    const storedState = parseCheckpoint(getStoredValue(workspaceSessionStorageKey(workspaceId)));
    const nextState = storedState ?? createFreshWorkspaceSession();

    replaceWorkspaceSession(state, nextState);

    const serialisedCheckpoint = serialiseCheckpoint(state);
    storeValue(workspaceSessionStorageKey(workspaceId), serialisedCheckpoint);
    entry.lastSerialisedCheckpoint = serialisedCheckpoint;
    entry.isInitialised = true;
  };

  return {
    activeTab,
    activeWorkspaceId,
    activateWorkspaceSession,
    getSelectedAgentName,
    isScratchpadOpen,
    prepareWorkspaceSession,
    resetWorkspaceSession,
    selectedAgentName,
    terminalActivePane,
    terminalZoomedPane
  };
});

const workspaceSessionStorageKey = (workspaceId: string) => `wade:workspace-session:${workspaceId}`;

const createFreshWorkspaceSession = (): WorkspaceSessionCheckpoint => ({
  activeTab: WorkspaceTabs.Terminal,
  isScratchpadOpen: false,
  terminal: {
    activePane: TerminalPanes.Agent,
    zoomedPane: null,
    selectedAgentName: ''
  },
  review: null
});

const createCheckpoint = (state: WorkspaceSessionCheckpoint): WorkspaceSessionCheckpoint => ({
  activeTab: state.activeTab,
  isScratchpadOpen: state.isScratchpadOpen,
  terminal: {
    activePane: state.terminal.activePane,
    zoomedPane: state.terminal.zoomedPane,
    selectedAgentName: state.terminal.selectedAgentName
  },
  review: null
});

const serialiseCheckpoint = (state: WorkspaceSessionCheckpoint) => JSON.stringify(createCheckpoint(state));

const replaceWorkspaceSession = (
  state: WorkspaceSessionCheckpoint,
  nextState: WorkspaceSessionCheckpoint
) => {
  state.activeTab = nextState.activeTab;
  state.isScratchpadOpen = nextState.isScratchpadOpen;
  state.terminal.activePane = nextState.terminal.activePane;
  state.terminal.zoomedPane = nextState.terminal.zoomedPane;
  state.terminal.selectedAgentName = nextState.terminal.selectedAgentName;
  state.review = null;
};

const isRecord = (value: unknown): value is Record<string, unknown> => (
  typeof value === 'object' && value !== null && !Array.isArray(value)
);

const isWorkspaceTab = (value: unknown): value is WorkspaceTab => (
  typeof value === 'string' && (workspaceTabs as readonly string[]).includes(value)
);

const isTerminalPane = (value: unknown): value is TerminalPaneId => (
  typeof value === 'string' && (terminalPanes as readonly string[]).includes(value)
);

const parseCheckpoint = (storedCheckpoint: string | null): WorkspaceSessionCheckpoint | undefined => {
  if (storedCheckpoint === null) {
    return undefined;
  }

  let value: unknown;
  try {
    value = JSON.parse(storedCheckpoint);
  } catch {
    return undefined;
  }

  if (!isRecord(value)
    || !isWorkspaceTab(value.activeTab)
    || typeof value.isScratchpadOpen !== 'boolean'
    || value.review !== null
    || !isRecord(value.terminal)
    || !isTerminalPane(value.terminal.activePane)
    || (value.terminal.zoomedPane !== null && !isTerminalPane(value.terminal.zoomedPane))
    || typeof value.terminal.selectedAgentName !== 'string') {
    return undefined;
  }

  return {
    activeTab: value.activeTab,
    isScratchpadOpen: value.isScratchpadOpen,
    terminal: {
      activePane: value.terminal.activePane,
      zoomedPane: value.terminal.zoomedPane,
      selectedAgentName: value.terminal.selectedAgentName
    },
    review: null
  };
};

const getStoredValue = (key: string) => {
  try {
    return window.localStorage.getItem(key);
  } catch {
    return null;
  }
};

const storeValue = (key: string, value: string) => {
  try {
    window.localStorage.setItem(key, value);
  } catch {
    return;
  }
};

const removeStoredValue = (key: string) => {
  try {
    window.localStorage.removeItem(key);
  } catch {
    return;
  }
};
