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
  const refreshRequests = new Map<string, Promise<Readonly<Workspace> | undefined>>();
  const refreshGenerations = new Map<string, number>();
  let linearConfigurationGeneration = 0;

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

    const requestGeneration = linearConfigurationGeneration;
    const workspaceState = ensureWorkspaceState(workspaceId);
    workspaceState.error = '';
    workspaceState.isLoading = true;

    const loadRequest = (async () => {
      try {
        const details = await getWorkspace(workspaceId);
        const issue =
          requestGeneration === linearConfigurationGeneration
            ? details.links.issue
            : (workspaceState.details?.links.issue ?? null);
        const detachedDetails = {
          ...details,
          links: {
            ...details.links,
            issue
          }
        };
        workspaceState.details = detachedDetails;

        return detachedDetails;
      } catch (requestError) {
        workspaceState.error = errorMessage(requestError);
        return undefined;
      } finally {
        loadRequests.delete(workspaceId);
        workspaceState.isLoading = refreshRequests.has(workspaceId);
      }
    })();

    loadRequests.set(workspaceId, loadRequest);
    return loadRequest;
  };

  const invalidateLinearIssueLinks = () => {
    linearConfigurationGeneration++;

    for (const workspaceState of workspaceStates.values()) {
      if (!workspaceState.details) {
        continue;
      }

      workspaceState.details = {
        ...workspaceState.details,
        links: {
          ...workspaceState.details.links,
          issue: null
        }
      };
    }
  };

  const refreshWorkspaceDetails = (workspaceId: string) => {
    refreshGenerations.set(workspaceId, linearConfigurationGeneration);

    const activeRefresh = refreshRequests.get(workspaceId);
    if (activeRefresh) {
      return activeRefresh;
    }

    const workspaceState = ensureWorkspaceState(workspaceId);
    workspaceState.isLoading = true;

    const refreshRequest = (async () => {
      let details: Readonly<Workspace> | undefined;

      while (true) {
        const requestedGeneration = refreshGenerations.get(workspaceId);
        const activeLoad = loadRequests.get(workspaceId);
        if (activeLoad) {
          await activeLoad;
        }

        details = await loadWorkspaceDetails(workspaceId);
        if (requestedGeneration === refreshGenerations.get(workspaceId)) {
          return details;
        }
      }
    })().finally(() => {
      refreshRequests.delete(workspaceId);
      workspaceState.isLoading = loadRequests.has(workspaceId);
    });

    refreshRequests.set(workspaceId, refreshRequest);
    return refreshRequest;
  };

  return {
    getWorkspaceDetails,
    getWorkspaceDetailsError,
    invalidateLinearIssueLinks,
    isWorkspaceDetailsLoading,
    loadWorkspaceDetails,
    refreshWorkspaceDetails
  };
});
