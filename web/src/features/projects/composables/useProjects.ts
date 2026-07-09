import { createSharedComposable, useStorage } from '@vueuse/core';
import { computed, readonly, ref } from 'vue';
import { listProjects } from '@/api/generated/wade';

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

export const useProjects = createSharedComposable(() => {
  const storedProjects = useStorage<string[]>(projectsStorageKey, [], localStorage, {
    serializer: projectsSerializer
  });
  const isSyncing = ref(false);
  const error = ref('');

  const projects = computed(() => normaliseProjects(storedProjects.value));

  let syncRequest: Promise<string[] | undefined> | undefined;

  const syncProjects = () => {
    if (syncRequest) {
      return syncRequest;
    }

    isSyncing.value = true;
    error.value = '';

    syncRequest = (async () => {
      try {
        const { projects } = await listProjects();
        const nextProjects = normaliseProjects(projects);
        storedProjects.value = nextProjects;

        return nextProjects;
      } catch (requestError) {
        error.value = requestError instanceof Error ? requestError.message : 'Projects request failed';
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
    projects: readonly(projects),
    syncProjects
  };
});
