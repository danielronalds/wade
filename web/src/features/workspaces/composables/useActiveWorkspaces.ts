import { createSharedComposable, useStorage } from '@vueuse/core';
import { computed, readonly, ref } from 'vue';
import { listWorkspaces, type WorkspaceSummary } from '@/api/generated/wade';

const activeWorkspacesStorageKey = 'wade:active-workspaces';

const normaliseActiveWorkspaces = (workspaces: unknown): WorkspaceSummary[] => {
  if (!Array.isArray(workspaces)) {
    return [];
  }

  const workspacesById = new Map<string, WorkspaceSummary>();
  for (const workspace of workspaces) {
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

const activeWorkspacesSerializer = {
  read: (value: string): WorkspaceSummary[] => {
    try {
      return normaliseActiveWorkspaces(JSON.parse(value));
    } catch {
      return [];
    }
  },
  write: (workspaces: WorkspaceSummary[]): string => JSON.stringify(normaliseActiveWorkspaces(workspaces))
};

export const useActiveWorkspaces = createSharedComposable(() => {
  const storedActiveWorkspaces = useStorage<WorkspaceSummary[]>(
    activeWorkspacesStorageKey,
    [],
    localStorage,
    { serializer: activeWorkspacesSerializer }
  );
  const isSyncing = ref(false);
  const error = ref('');

  const activeWorkspaces = computed(() => normaliseActiveWorkspaces(storedActiveWorkspaces.value));

  let syncRequest: Promise<WorkspaceSummary[] | undefined> | undefined;

  const syncActiveWorkspaces = () => {
    if (syncRequest) {
      return syncRequest;
    }

    isSyncing.value = true;
    error.value = '';

    syncRequest = (async () => {
      try {
        const { items } = await listWorkspaces({ activity: 'active' });
        storedActiveWorkspaces.value = normaliseActiveWorkspaces(items);

        return activeWorkspaces.value;
      } catch (requestError) {
        error.value = requestError instanceof Error ? requestError.message : 'Active workspace request failed';
        return undefined;
      } finally {
        isSyncing.value = false;
        syncRequest = undefined;
      }
    })();

    return syncRequest;
  };

  return {
    activeWorkspaces: readonly(activeWorkspaces),
    error: readonly(error),
    isSyncing: readonly(isSyncing),
    syncActiveWorkspaces
  };
});
