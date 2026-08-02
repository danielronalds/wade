import { createSharedComposable, useStorage } from '@vueuse/core';
import { computed, readonly, ref } from 'vue';
import { listWorkspaces, type WorkspaceSummary } from '@/api/generated/wade';

const workspacesStorageKey = 'wade:workspaces';
const legacyProjectsStorageKey = 'wade:projects';

const legacyWorkspaceSummary = (workspaceId: string): WorkspaceSummary => ({
  activity: { activeTerminalCount: 0 },
  branch: null,
  id: workspaceId,
  links: {
    issue: null,
    pullRequest: null,
    repository: null
  },
  name: workspaceId,
  remoteRepositoryId: null,
  repositoryId: null,
  worktree: null
});

const normaliseWorkspaces = (workspaces: unknown): WorkspaceSummary[] => {
  if (!Array.isArray(workspaces)) {
    return [];
  }

  const workspacesById = new Map<string, WorkspaceSummary>();
  for (const workspace of workspaces) {
    if (typeof workspace === 'string' && workspace.length > 0) {
      workspacesById.set(workspace, legacyWorkspaceSummary(workspace));
      continue;
    }
    if (!workspace || typeof workspace !== 'object') {
      continue;
    }

    const candidate = workspace as Partial<WorkspaceSummary>;
    if (typeof candidate.id === 'string' && candidate.id.length > 0 && typeof candidate.name === 'string') {
      workspacesById.set(candidate.id, candidate as WorkspaceSummary);
    }
  }

  return Array.from(workspacesById.values()).sort((firstWorkspace, secondWorkspace) => (
    firstWorkspace.name.localeCompare(secondWorkspace.name)
    || firstWorkspace.id.localeCompare(secondWorkspace.id)
  ));
};

const workspacesSerializer = {
  read: (value: string): WorkspaceSummary[] => {
    try {
      return normaliseWorkspaces(JSON.parse(value));
    } catch {
      return [];
    }
  },
  write: (workspaces: WorkspaceSummary[]): string => JSON.stringify(normaliseWorkspaces(workspaces))
};

export const useWorkspaces = createSharedComposable(() => {
  const legacyProjects = localStorage.getItem(legacyProjectsStorageKey);
  const initialWorkspaces = legacyProjects === null ? [] : workspacesSerializer.read(legacyProjects);
  const storedWorkspaces = useStorage<WorkspaceSummary[]>(workspacesStorageKey, initialWorkspaces, localStorage, {
    serializer: workspacesSerializer
  });
  localStorage.removeItem(legacyProjectsStorageKey);

  const isSyncing = ref(false);
  const error = ref('');
  const workspaces = computed(() => normaliseWorkspaces(storedWorkspaces.value));

  let syncRequest: Promise<WorkspaceSummary[] | undefined> | undefined;

  const syncWorkspaces = () => {
    if (syncRequest) {
      return syncRequest;
    }

    isSyncing.value = true;
    error.value = '';

    syncRequest = (async () => {
      try {
        const { items } = await listWorkspaces();
        const nextWorkspaces = normaliseWorkspaces(items);
        storedWorkspaces.value = nextWorkspaces;

        return nextWorkspaces;
      } catch (requestError) {
        error.value = requestError instanceof Error ? requestError.message : 'Workspace request failed';
        return undefined;
      } finally {
        isSyncing.value = false;
        syncRequest = undefined;
      }
    })();

    return syncRequest;
  };

  return {
    error: readonly(error),
    isSyncing: readonly(isSyncing),
    workspaces: readonly(workspaces),
    syncWorkspaces
  };
});
