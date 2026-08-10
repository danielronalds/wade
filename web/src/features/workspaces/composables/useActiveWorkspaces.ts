import { createSharedComposable, useStorage } from '@vueuse/core';
import { readonly, ref } from 'vue';
import { listWorkspaces, type WorkspaceSummary } from '@/api/generated/wade';

const activeWorkspacesStorageKey = 'wade:active-workspaces';

export const useActiveWorkspaces = createSharedComposable(() => {
  const storedActiveWorkspaces = useStorage<WorkspaceSummary[]>(activeWorkspacesStorageKey, [], localStorage);
  const activeWorkspaces = readonly(storedActiveWorkspaces);
  const isSyncing = ref(false);
  const error = ref('');

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
        const nextActiveWorkspaces = sortActiveWorkspaces(items);
        storedActiveWorkspaces.value = nextActiveWorkspaces;

        return nextActiveWorkspaces;
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
    activeWorkspaces,
    error: readonly(error),
    isSyncing: readonly(isSyncing),
    syncActiveWorkspaces
  };
});

const sortActiveWorkspaces = (workspaces: WorkspaceSummary[]): WorkspaceSummary[] =>
  [...workspaces].sort(
    (firstWorkspace, secondWorkspace) =>
      firstWorkspace.name.localeCompare(secondWorkspace.name) || firstWorkspace.id.localeCompare(secondWorkspace.id)
  );
