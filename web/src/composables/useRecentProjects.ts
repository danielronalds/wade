import { createSharedComposable, useStorage } from '@vueuse/core';
import { computed, readonly } from 'vue';

const recentProjectsStorageKey = 'web-terminal:recent-projects';
const recentProjectsLimit = 5;

const normaliseRecentProjects = (projects: unknown): string[] => {
  if (!Array.isArray(projects)) {
    return [];
  }

  return projects
    .filter((project): project is string => typeof project === 'string' && project.length > 0)
    .slice(0, recentProjectsLimit);
};

const recentProjectsSerializer = {
  read: (value: string): string[] => {
    try {
      return normaliseRecentProjects(JSON.parse(value));
    } catch {
      return [];
    }
  },
  write: (projects: string[]): string => JSON.stringify(normaliseRecentProjects(projects))
};

export const useRecentProjects = createSharedComposable(() => {
  const storedRecentProjects = useStorage<string[]>(recentProjectsStorageKey, [], localStorage, {
    serializer: recentProjectsSerializer
  });

  const recentProjects = computed(() => normaliseRecentProjects(storedRecentProjects.value));

  const recordRecentProject = (projectName: string) => {
    if (projectName.length === 0) {
      return;
    }

    const nextRecentProjects = normaliseRecentProjects([
      projectName,
      ...recentProjects.value.filter((project) => project !== projectName)
    ]);

    storedRecentProjects.value = nextRecentProjects;
  };

  return {
    recentProjects: readonly(recentProjects),
    recordRecentProject
  };
});
