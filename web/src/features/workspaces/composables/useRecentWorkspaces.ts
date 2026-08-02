import { createSharedComposable, useStorage } from '@vueuse/core';
import { computed, readonly } from 'vue';
import type { WorkspaceSummary } from '@/api/generated/wade';

const recentWorkspacesStorageKey = 'wade:recent-workspaces';
const legacyRecentProjectsStorageKey = 'wade:recent-projects';
const recentWorkspacesLimit = 5;

const normaliseRecentWorkspaceIds = (workspaceIds: unknown): string[] => {
  if (!Array.isArray(workspaceIds)) {
    return [];
  }

  return workspaceIds
    .filter((workspaceId): workspaceId is string => typeof workspaceId === 'string' && workspaceId.length > 0)
    .slice(0, recentWorkspacesLimit);
};

const recentWorkspacesSerializer = {
  read: (value: string): string[] => {
    try {
      return normaliseRecentWorkspaceIds(JSON.parse(value));
    } catch {
      return [];
    }
  },
  write: (workspaceIds: string[]): string => JSON.stringify(normaliseRecentWorkspaceIds(workspaceIds))
};

export const useRecentWorkspaces = createSharedComposable(() => {
  const legacyRecentProjects = localStorage.getItem(legacyRecentProjectsStorageKey);
  const initialWorkspaceIds = legacyRecentProjects === null
    ? []
    : recentWorkspacesSerializer.read(legacyRecentProjects);
  const storedRecentWorkspaceIds = useStorage<string[]>(
    recentWorkspacesStorageKey,
    initialWorkspaceIds,
    localStorage,
    { serializer: recentWorkspacesSerializer }
  );
  localStorage.removeItem(legacyRecentProjectsStorageKey);

  const recentWorkspaceIds = computed(() => normaliseRecentWorkspaceIds(storedRecentWorkspaceIds.value));

  const recordRecentWorkspace = (workspaceId: string) => {
    if (workspaceId.length === 0) {
      return;
    }

    storedRecentWorkspaceIds.value = normaliseRecentWorkspaceIds([
      workspaceId,
      ...recentWorkspaceIds.value.filter((recentWorkspaceId) => recentWorkspaceId !== workspaceId)
    ]);
  };

  const removeUnavailableRecentWorkspaces = (availableWorkspaces: readonly WorkspaceSummary[]) => {
    const availableWorkspaceIds = new Set(availableWorkspaces.map((workspace) => workspace.id));
    storedRecentWorkspaceIds.value = recentWorkspaceIds.value.filter((workspaceId) => (
      availableWorkspaceIds.has(workspaceId)
    ));
  };

  return {
    recentWorkspaceIds: readonly(recentWorkspaceIds),
    recordRecentWorkspace,
    removeUnavailableRecentWorkspaces
  };
});
