import { createSharedComposable } from '@vueuse/core';
import { computed, readonly, ref } from 'vue';
import { listWorkspaces, type WorkspaceSummary } from '@/api/generated/wade';

const normaliseActiveWorkspaces = (workspaces: WorkspaceSummary[]): WorkspaceSummary[] => (
  [...workspaces].sort((firstWorkspace, secondWorkspace) => (
    firstWorkspace.name.localeCompare(secondWorkspace.name)
    || firstWorkspace.id.localeCompare(secondWorkspace.id)
  ))
);

export const useActiveWorkspaces = createSharedComposable(() => {
  const storedActiveWorkspaces = ref<WorkspaceSummary[]>([]);
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
