import { createSharedComposable, useStorage } from '@vueuse/core';
import { readonly, ref } from 'vue';
import { listWorkspaces, type WorkspaceSummary } from '@/api/generated/wade';

const workspacesStorageKey = 'wade:workspaces';

export const useWorkspaces = createSharedComposable(() => {
  const storedWorkspaces = useStorage<WorkspaceSummary[]>(workspacesStorageKey, [], localStorage);

  const workspaces = readonly(storedWorkspaces);
  const isSyncing = ref(false);
  const error = ref('');

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
        const nextWorkspaces = sortWorkspaces(items);
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
    workspaces,
    syncWorkspaces
  };
});

const sortWorkspaces = (workspaces: WorkspaceSummary[]): WorkspaceSummary[] => (
  [...workspaces].sort((firstWorkspace, secondWorkspace) => (
    firstWorkspace.name.localeCompare(secondWorkspace.name)
    || firstWorkspace.id.localeCompare(secondWorkspace.id)
  ))
);
