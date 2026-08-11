import { defineStore } from 'pinia';
import { reactive } from 'vue';
import { getWorkspace, type Workspace } from '@/api/generated/wade';

type WorkspaceDetailsState = {
  details?: Readonly<Workspace>;
  error: string;
  isLoading: boolean;
};

const errorMessage = (error: unknown) => (error instanceof Error ? error.message : 'Workspace details request failed');

export const useWorkspaceDetailsStore = defineStore('workspace-details', () => {
  const workspaceStates = reactive(new Map<string, WorkspaceDetailsState>());
  const loadRequests = new Map<string, Promise<Readonly<Workspace> | undefined>>();

  const ensureWorkspaceState = (workspaceId: string) => {
    const existingState = workspaceStates.get(workspaceId);
    if (existingState) {
      return existingState;
    }

    workspaceStates.set(workspaceId, {
      error: '',
      isLoading: false
    });

    return workspaceStates.get(workspaceId)!;
  };

  const getWorkspaceDetails = (workspaceId: string) => workspaceStates.get(workspaceId)?.details;
  const getWorkspaceDetailsError = (workspaceId: string) => workspaceStates.get(workspaceId)?.error ?? '';
  const isWorkspaceDetailsLoading = (workspaceId: string) => workspaceStates.get(workspaceId)?.isLoading ?? false;

  const loadWorkspaceDetails = (workspaceId: string) => {
    const activeRequest = loadRequests.get(workspaceId);
    if (activeRequest) {
      return activeRequest;
    }

    const workspaceState = ensureWorkspaceState(workspaceId);
    workspaceState.error = '';
    workspaceState.isLoading = true;

    const loadRequest = (async () => {
      try {
        const details = await getWorkspace(workspaceId);
        workspaceState.details = details;

        return details;
      } catch (requestError) {
        workspaceState.error = errorMessage(requestError);
        return undefined;
      } finally {
        workspaceState.isLoading = false;
        loadRequests.delete(workspaceId);
      }
    })();

    loadRequests.set(workspaceId, loadRequest);
    return loadRequest;
  };

  return {
    getWorkspaceDetails,
    getWorkspaceDetailsError,
    isWorkspaceDetailsLoading,
    loadWorkspaceDetails
  };
});
