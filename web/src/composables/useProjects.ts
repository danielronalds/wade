import { createSharedComposable, useStorage } from '@vueuse/core';
import { computed, readonly, ref } from 'vue';

type ProjectsResponse = {
  projects: string[];
};

const projectsStorageKey = 'wade:projects';

const normaliseProjects = (projects: unknown): string[] => {
  if (!Array.isArray(projects)) {
    return [];
  }

  return Array.from(new Set(
    projects.filter((project): project is string => typeof project === 'string' && project.length > 0)
  )).sort((firstProject, secondProject) => firstProject.localeCompare(secondProject));
};

const projectsSerializer = {
  read: (value: string): string[] => {
    try {
      return normaliseProjects(JSON.parse(value));
    } catch {
      return [];
    }
  },
  write: (projects: string[]): string => JSON.stringify(normaliseProjects(projects))
};

const isProjectsResponse = (value: unknown): value is ProjectsResponse => {
  if (!value || typeof value !== 'object') {
    return false;
  }

  const response = value as Partial<ProjectsResponse>;

  return Array.isArray(response.projects)
    && response.projects.every((project) => typeof project === 'string');
};

export const useProjects = createSharedComposable(() => {
  const storedProjects = useStorage<string[]>(projectsStorageKey, [], localStorage, {
    serializer: projectsSerializer
  });
  const isSyncing = ref(false);
  const error = ref('');

  const projects = computed(() => normaliseProjects(storedProjects.value));

  const syncProjects = async () => {
    if (isSyncing.value) {
      return;
    }

    isSyncing.value = true;
    error.value = '';

    try {
      const response = await fetch('/api/projects');

      if (!response.ok) {
        throw new Error(`Projects request failed with ${response.status}`);
      }

      const projectsResponse: unknown = await response.json();
      if (!isProjectsResponse(projectsResponse)) {
        throw new Error('Projects response was invalid');
      }

      storedProjects.value = normaliseProjects(projectsResponse.projects);
    } catch (requestError) {
      error.value = requestError instanceof Error ? requestError.message : 'Projects request failed';
    } finally {
      isSyncing.value = false;
    }
  };

  return {
    error: readonly(error),
    isSyncing: readonly(isSyncing),
    projects: readonly(projects),
    syncProjects
  };
});
